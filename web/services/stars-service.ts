import {
  getUnauthedData,
  STARS_CONFIG,
  STARS_WALLET,
  STARS_PAYPAL_ORDER,
  STARS_PAYPAL_CAPTURE,
  STARS_SEND,
} from '../utils/apis';
import { StarSettings, StarWalletSummary, StarSendEvent } from '../interfaces/stars.model';

function withAccessToken(url: string, accessToken: string): string {
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}accessToken=${encodeURIComponent(accessToken)}`;
}

export class StarsService {
  public static async getConfig(): Promise<StarSettings> {
    return getUnauthedData(STARS_CONFIG);
  }

  public static async getWallet(accessToken: string): Promise<StarWalletSummary> {
    return getUnauthedData(withAccessToken(STARS_WALLET, accessToken));
  }

  public static async createPayPalOrder(accessToken: string, packageId: number) {
    return getUnauthedData(withAccessToken(STARS_PAYPAL_ORDER, accessToken), {
      method: 'POST',
      data: { packageId },
    });
  }

  public static async capturePayPalOrder(accessToken: string, paypalOrderId: string) {
    return getUnauthedData(withAccessToken(STARS_PAYPAL_CAPTURE, accessToken), {
      method: 'POST',
      data: { paypalOrderId },
    });
  }

  public static async sendStars(
    accessToken: string,
    amount: number,
    message: string,
    effect: string,
  ): Promise<StarSendEvent> {
    return getUnauthedData(withAccessToken(STARS_SEND, accessToken), {
      method: 'POST',
      data: { amount, message, effect },
    });
  }
}
