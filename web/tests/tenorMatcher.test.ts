import { renderTenorGifEmbeds } from '../components/chat/ChatUserMessage/tenorMatcher';

describe('renderTenorGifEmbeds', () => {
  test('turns sanitized Tenor links into inline GIF embeds', () => {
    const url = 'https://media1.tenor.com/m/example/tenor.gif';
    const content = `<a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a>`;

    const rendered = renderTenorGifEmbeds(content);

    expect(rendered).toContain('class="chat-tenor-gif-link"');
    expect(rendered).toContain('class="chat-tenor-gif"');
    expect(rendered).toContain(`src="${url}"`);
    expect(rendered).toContain('alt="Tenor GIF"');
  });

  test('turns Android Markdown Tenor messages into inline GIF embeds', () => {
    const url = 'https://media1.tenor.com/m/example/tenor.gif';
    const rendered = renderTenorGifEmbeds(`hello ![Tenor GIF](${url})`);

    expect(rendered).toContain('hello ');
    expect(rendered).toContain('class="chat-tenor-gif-link"');
    expect(rendered).toContain(`href="${url}"`);
    expect(rendered).toContain(`src="${url}"`);
    expect(rendered).not.toContain('![Tenor GIF]');
  });
});
