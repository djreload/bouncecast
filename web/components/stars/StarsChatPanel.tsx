import { Alert, Button, Input, InputNumber, List, Modal, Select, Space, Tabs, message } from 'antd';
import classNames from 'classnames';
import { FC, useEffect, useRef, useState } from 'react';
import { useRecoilValue } from 'recoil';
import { accessTokenAtom } from '../stores/ClientConfigStore';
import { StarPackage, StarSettings, StarWalletSummary } from '../../interfaces/stars.model';
import { StarsService } from '../../services/stars-service';
import styles from './StarsChatPanel.module.scss';

declare global {
  interface Window {
    paypal?: any;
  }
}

const effectOptions = [
  { label: 'Sparkle', value: 'sparkle' },
  { label: 'Fireworks', value: 'fireworks' },
  { label: 'Hearts', value: 'hearts' },
  { label: 'Hype', value: 'hype' },
  { label: 'DJ drop', value: 'dj_drop' },
];

function formatPrice(pkg: StarPackage): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: pkg.currency,
  }).format(pkg.priceCents / 100);
}

function loadPayPalScript(config: StarSettings): Promise<void> {
  if (window.paypal) {
    return Promise.resolve();
  }

  return new Promise((resolve, reject) => {
    const existingScript = document.querySelector<HTMLScriptElement>('script[data-bouncecast-paypal]');
    if (existingScript) {
      existingScript.addEventListener('load', () => resolve());
      existingScript.addEventListener('error', reject);
      return;
    }

    const script = document.createElement('script');
    script.dataset.bouncecastPaypal = 'true';
    script.src = `https://www.paypal.com/sdk/js?client-id=${encodeURIComponent(
      config.paypalClientId,
    )}&currency=${encodeURIComponent(config.currency)}&intent=capture`;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = reject;
    document.body.appendChild(script);
  });
}

export const StarsChatPanel: FC = () => {
  const accessToken = useRecoilValue<string>(accessTokenAtom);
  const [config, setConfig] = useState<StarSettings>(null);
  const [wallet, setWallet] = useState<StarWalletSummary>(null);
  const [buyOpen, setBuyOpen] = useState(false);
  const [sendOpen, setSendOpen] = useState(false);
  const [selectedPackageId, setSelectedPackageId] = useState<number>(null);
  const [sendAmount, setSendAmount] = useState<number>(100);
  const [sendMessage, setSendMessage] = useState('');
  const [effect, setEffect] = useState('sparkle');
  const [payPalReady, setPayPalReady] = useState(false);
  const paypalRef = useRef<HTMLDivElement>(null);

  const refreshWallet = async () => {
    if (!accessToken) {
      return;
    }
    try {
      setWallet(await StarsService.getWallet(accessToken));
    } catch (error) {
      console.error(error);
    }
  };

  useEffect(() => {
    StarsService.getConfig()
      .then(nextConfig => {
        setConfig(nextConfig);
        setSelectedPackageId(nextConfig.packages?.[0]?.id || null);
      })
      .catch(error => console.error(error));
  }, []);

  useEffect(() => {
    if (config?.enabled) {
      refreshWallet();
    }
  }, [accessToken, config?.enabled]);

  useEffect(() => {
    if (!buyOpen || !config?.paypalClientId || !selectedPackageId || !paypalRef.current) {
      return;
    }

    let cancelled = false;
    paypalRef.current.innerHTML = '';
    loadPayPalScript(config)
      .then(() => {
        if (cancelled || !window.paypal || !paypalRef.current) {
          return;
        }
        setPayPalReady(true);
        window.paypal
          .Buttons({
            createOrder: async () => {
              const result = await StarsService.createPayPalOrder(accessToken, selectedPackageId);
              return result.paypalOrderId;
            },
            onApprove: async data => {
              await StarsService.capturePayPalOrder(accessToken, data.orderID);
              message.success('Stars added to your wallet');
              setBuyOpen(false);
              refreshWallet();
            },
            onError: error => {
              console.error(error);
              message.error('PayPal checkout failed');
            },
          })
          .render(paypalRef.current);
      })
      .catch(error => {
        console.error(error);
        message.error('Unable to load PayPal checkout');
      });

    return () => {
      cancelled = true;
    };
  }, [buyOpen, selectedPackageId, config?.paypalClientId]);

  if (!config?.enabled || !accessToken) {
    return null;
  }

  const packages = config.packages || [];
  const balance = wallet?.wallet?.balance || 0;

  const handleSendStars = async () => {
    try {
      await StarsService.sendStars(accessToken, sendAmount, sendMessage, effect);
      message.success('Stars sent');
      setSendOpen(false);
      setSendMessage('');
      refreshWallet();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to send Stars');
    }
  };

  return (
    <>
      <button
        type="button"
        className={styles.starsButton}
        title="Stars"
        aria-label="Stars"
        onClick={() => setBuyOpen(true)}
      >
        {balance} Stars
      </button>
      <Modal
        title="Stars"
        open={buyOpen}
        onCancel={() => setBuyOpen(false)}
        footer={null}
        destroyOnClose
      >
        <Tabs
          items={[
            {
              key: 'buy',
              label: 'Buy',
              children: (
                <>
                  <p className={styles.modalIntro}>{config.supportMessage}</p>
                  <div className={styles.packageList}>
                    {packages.map(pkg => (
                      <button
                        key={pkg.id}
                        type="button"
                        className={classNames(styles.packageButton, {
                          [styles.selectedPackage]: selectedPackageId === pkg.id,
                        })}
                        onClick={() => {
                          setPayPalReady(false);
                          setSelectedPackageId(pkg.id);
                        }}
                      >
                        <strong>{pkg.name}</strong>
                        <span>{formatPrice(pkg)}</span>
                      </button>
                    ))}
                  </div>
                  {!config.paypalClientId && (
                    <Alert type="warning" message="PayPal is not configured yet." showIcon />
                  )}
                  <div ref={paypalRef} className={styles.paypalContainer}>
                    {config.paypalClientId && !payPalReady && <Alert message="Loading PayPal" />}
                  </div>
                </>
              ),
            },
            {
              key: 'send',
              label: 'Send',
              children: (
                <div className={styles.sendGrid}>
                  <Alert message={`Balance: ${balance} Stars`} type="info" showIcon />
                  <InputNumber
                    min={config.minimumSendAmount}
                    max={Math.min(config.maximumSendAmount, Math.max(balance, 0))}
                    value={sendAmount}
                    onChange={value => setSendAmount(Number(value || config.minimumSendAmount))}
                    style={{ width: '100%' }}
                  />
                  <Input
                    maxLength={120}
                    value={sendMessage}
                    placeholder="Short message"
                    onChange={event => setSendMessage(event.target.value)}
                  />
                  <Select value={effect} options={effectOptions} onChange={setEffect} />
                  <Space>
                    <Button type="primary" onClick={handleSendStars} disabled={balance < sendAmount}>
                      Send Stars
                    </Button>
                    <Button onClick={() => setSendOpen(true)}>History</Button>
                  </Space>
                </div>
              ),
            },
          ]}
        />
      </Modal>
      <Modal
        title="Star history"
        open={sendOpen}
        onCancel={() => setSendOpen(false)}
        footer={null}
      >
        <List
          dataSource={wallet?.transactions || []}
          locale={{ emptyText: 'No Star transactions yet' }}
          renderItem={item => (
            <List.Item>
              <div className={styles.walletLine}>
                <span>{item.transactionType}</span>
                <strong>{item.amount > 0 ? `+${item.amount}` : item.amount}</strong>
              </div>
            </List.Item>
          )}
        />
      </Modal>
    </>
  );
};
