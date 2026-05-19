package handlers

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/owncast/owncast/models"
	"github.com/owncast/owncast/utils"
)

type normalizedBounceCastProfile struct {
	avatarURL       string
	bio             string
	genresJSON      string
	socialLinksJSON string
	heroImageURL    string
}

func normalizeBounceCastDJProfileFields(avatarURL string, bio string, genres []string, socialLinks []models.BounceCastSocialLink, heroImageURL string) (normalizedBounceCastProfile, error) {
	normalizedAvatarURL, err := normalizeBounceCastProfileImageURL(avatarURL)
	if err != nil {
		return normalizedBounceCastProfile{}, err
	}
	normalizedHeroImageURL, err := normalizeBounceCastProfileImageURL(heroImageURL)
	if err != nil {
		return normalizedBounceCastProfile{}, err
	}
	normalizedGenres, err := normalizeBounceCastGenres(genres)
	if err != nil {
		return normalizedBounceCastProfile{}, err
	}
	normalizedLinks, err := normalizeBounceCastSocialLinks(socialLinks)
	if err != nil {
		return normalizedBounceCastProfile{}, err
	}

	genresJSON, err := marshalBounceCastProfileJSON(normalizedGenres)
	if err != nil {
		return normalizedBounceCastProfile{}, err
	}
	socialLinksJSON, err := marshalBounceCastProfileJSON(normalizedLinks)
	if err != nil {
		return normalizedBounceCastProfile{}, err
	}

	return normalizedBounceCastProfile{
		avatarURL:       normalizedAvatarURL,
		bio:             utils.MakeSafeStringOfLength(bio, 600),
		genresJSON:      genresJSON,
		socialLinksJSON: socialLinksJSON,
		heroImageURL:    normalizedHeroImageURL,
	}, nil
}

func normalizeBounceCastGenres(values []string) ([]string, error) {
	genres := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		genre := utils.MakeSafeStringOfLength(value, 32)
		if genre == "" {
			continue
		}
		key := strings.ToLower(genre)
		if seen[key] {
			continue
		}
		seen[key] = true
		genres = append(genres, genre)
		if len(genres) > 8 {
			return nil, errors.New("genres must contain 8 items or fewer")
		}
	}
	return genres, nil
}

func normalizeBounceCastSocialLinks(values []models.BounceCastSocialLink) ([]models.BounceCastSocialLink, error) {
	links := []models.BounceCastSocialLink{}
	for _, value := range values {
		label := utils.MakeSafeStringOfLength(value.Label, 40)
		linkURL := strings.TrimSpace(value.URL)
		if label == "" && linkURL == "" {
			continue
		}
		if label == "" || linkURL == "" {
			return nil, errors.New("social links require both label and URL")
		}
		if len(links) >= 6 {
			return nil, errors.New("social links must contain 6 items or fewer")
		}
		parsedURL, err := url.Parse(linkURL)
		if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
			return nil, errors.New("social link URLs must be valid http(s) URLs")
		}
		links = append(links, models.BounceCastSocialLink{
			Label: label,
			URL:   parsedURL.String(),
		})
	}
	return links, nil
}

func marshalBounceCastProfileJSON(value interface{}) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	if string(body) == "[]" || string(body) == "null" {
		return "", nil
	}
	return string(body), nil
}

func parseBounceCastGenres(value string) []string {
	var genres []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &genres); err == nil {
		normalized, _ := normalizeBounceCastGenres(genres)
		return normalized
	}
	normalized, _ := normalizeBounceCastGenres(strings.Split(value, ","))
	return normalized
}

func parseBounceCastSocialLinks(value string) []models.BounceCastSocialLink {
	var links []models.BounceCastSocialLink
	if err := json.Unmarshal([]byte(strings.TrimSpace(value)), &links); err != nil {
		return []models.BounceCastSocialLink{}
	}
	normalized, err := normalizeBounceCastSocialLinks(links)
	if err != nil {
		return []models.BounceCastSocialLink{}
	}
	return normalized
}
