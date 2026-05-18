import React, { useEffect } from 'react';
import { act, render, screen } from '@testing-library/react';
import { RecoilRoot, useSetRecoilState } from 'recoil';
import { StarsOverlay } from '../components/stars/StarsOverlay';
import { starsOverlayEventsAtom } from '../components/stores/ClientConfigStore';
import { MessageType, StarsSentSocketEvent } from '../interfaces/socket-events';

jest.mock(
  '../components/stars/StarsOverlay.module.scss',
  () =>
    new Proxy(
      {},
      {
        get: (_, property) => String(property),
      },
    ),
);

declare global {
  interface Window {
    pushStarsOverlayEvent: (event: StarsSentSocketEvent) => void;
  }
}

const baseEvent = {
  timestamp: new Date(),
  type: MessageType.STARS_SENT,
  displayName: 'DJ Test',
  effect: 'sparkle',
  soundEnabled: false,
};

const StarsOverlayHarness = () => {
  const setStarsOverlayEvents = useSetRecoilState(starsOverlayEventsAtom);

  useEffect(() => {
    window.pushStarsOverlayEvent = (event: StarsSentSocketEvent) => {
      setStarsOverlayEvents(currentEvents => [...currentEvents, event]);
    };
  }, [setStarsOverlayEvents]);

  return <StarsOverlay />;
};

describe('StarsOverlay', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    act(() => {
      jest.runOnlyPendingTimers();
    });
    jest.useRealTimers();
    delete window.pushStarsOverlayEvent;
  });

  it('shows queued star events after a previous overlay is dismissed', () => {
    render(
      <RecoilRoot>
        <StarsOverlayHarness />
      </RecoilRoot>,
    );

    act(() => {
      window.pushStarsOverlayEvent({
        ...baseEvent,
        id: 'stars-1',
        amount: 100,
        message: 'First drop',
      });
    });

    expect(screen.getByText(/100 Stars/)).toBeInTheDocument();

    act(() => {
      window.pushStarsOverlayEvent({
        ...baseEvent,
        id: 'stars-2',
        amount: 200,
        message: 'Second drop',
      });
    });

    act(() => {
      jest.advanceTimersByTime(4200);
    });

    expect(screen.getByText(/200 Stars/)).toBeInTheDocument();
    expect(screen.getByText('Second drop')).toBeInTheDocument();
  });

  it('renders the selected effect layer', () => {
    render(
      <RecoilRoot>
        <StarsOverlayHarness />
      </RecoilRoot>,
    );

    act(() => {
      window.pushStarsOverlayEvent({
        ...baseEvent,
        id: 'stars-fireworks',
        amount: 300,
        effect: 'fireworks',
        message: 'Light it up',
      });
    });

    expect(screen.getByTestId('stars-overlay-toast')).toHaveAttribute('data-effect', 'fireworks');
    expect(document.querySelectorAll('[data-effect-particle="fireworks"]')).toHaveLength(24);
  });
});
