/* eslint-disable class-methods-use-this */
import { ChildrenNode, Matcher, MatchResponse, Node } from 'interweave';
import React from 'react';

export interface ChatMessageTenorGifProps {
  key: string;
  url: string;
}

const tenorGifURLRegex = /https:\/\/media\d*\.tenor\.com\/[^\s<>()"]+\.gif(?:\?[^\s<>()"]*)?/i;

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
        rel: 'noopener noreferrer',
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
