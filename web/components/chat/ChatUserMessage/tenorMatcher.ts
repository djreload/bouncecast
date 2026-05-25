/* eslint-disable class-methods-use-this */
import { ChildrenNode, Matcher, MatchResponse, Node } from 'interweave';
import React from 'react';

export interface ChatMessageTenorGifProps {
  key: string;
  url: string;
}

const tenorGifURLPattern = String.raw`https:\/\/media\d*\.tenor\.com\/[^\s<>()"]+\.gif(?:\?[^\s<>()"]*)?`;
const tenorGifURLRegex = new RegExp(tenorGifURLPattern, 'i');
const tenorGifMarkdownRegex = new RegExp(String.raw`!\[[^\]]*]\((${tenorGifURLPattern})\)`, 'gi');
const tenorGifAnchorRegex = new RegExp(
  String.raw`<a\b[^>]*href="(${tenorGifURLPattern})"[^>]*>[\s\S]*?<\/a>`,
  'gi',
);

function renderTenorGifLink(url: string): string {
  const safeUrl = String(url);
  return `<a class="chat-tenor-gif-link" href="${safeUrl}" target="_blank" rel="noreferrer"><img alt="Tenor GIF" class="chat-tenor-gif" loading="lazy" src="${safeUrl}" /></a>`;
}

export function renderTenorGifEmbeds(content: string): string {
  return (content || '')
    .replace(tenorGifAnchorRegex, (_match, url) => renderTenorGifLink(url))
    .replace(tenorGifMarkdownRegex, (_match, url) => renderTenorGifLink(url));
}

export class ChatMessageTenorGifMatcher extends Matcher<ChatMessageTenorGifProps> {
  match(str: string): MatchResponse<Partial<ChatMessageTenorGifProps>> | null {
    const result = str.match(tenorGifURLRegex);

    if (!result) {
      return null;
    }

    return {
      index: result.index!,
      length: result[0].length,
      match: result[0],
      valid: true,
      url: result[0],
    };
  }

  replaceWith(_children: ChildrenNode, props: ChatMessageTenorGifProps): Node {
    const { key, url } = props;
    return React.createElement(
      'a',
      {
        key,
        className: 'chat-tenor-gif-link',
        href: url,
        target: '_blank',
        rel: 'noreferrer',
      },
      React.createElement('img', {
        alt: 'GIF reaction',
        className: 'chat-tenor-gif',
        loading: 'lazy',
        src: url,
      }),
    );
  }

  asTag(): string {
    return 'a';
  }
}
