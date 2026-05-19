/* eslint-disable @next/next/no-css-tags */
import React, { useState, useEffect, useContext, ReactElement } from 'react';
import { Skeleton, Card, Statistic, Row, Col } from 'antd';
import { formatDistanceToNow, formatRelative } from 'date-fns';
import dynamic from 'next/dynamic';
import { useTranslation } from 'next-export-i18n';
import { ServerStatusContext } from '../../utils/server-status-context';
import { LogTable } from '../../components/admin/LogTable';
import { Offline } from '../../components/admin/Offline';
import { StreamHealthOverview } from '../../components/admin/StreamHealthOverview';

import { BOUNCECAST_COMMAND_CENTER, LOGS_WARN, fetchData, FETCH_INTERVAL } from '../../utils/apis';
import { formatIPAddress, isEmptyObject } from '../../utils/format';
import { NewsFeed } from '../../components/admin/NewsFeed';

import { AdminLayout } from '../../components/layouts/AdminLayout';

// Lazy loaded components

const UserOutlined = dynamic(() => import('@ant-design/icons/UserOutlined'), {
  ssr: false,
});

const ClockCircleOutlined = dynamic(() => import('@ant-design/icons/ClockCircleOutlined'), {
  ssr: false,
});

const TeamOutlined = dynamic(() => import('@ant-design/icons/TeamOutlined'), {
  ssr: false,
});

const CalendarOutlined = dynamic(() => import('@ant-design/icons/CalendarOutlined'), {
  ssr: false,
});

const StarOutlined = dynamic(() => import('@ant-design/icons/StarOutlined'), {
  ssr: false,
});

const SafetyCertificateOutlined = dynamic(
  () => import('@ant-design/icons/SafetyCertificateOutlined'),
  { ssr: false },
);

type CommandCenterSummary = {
  accounts: {
    total: number;
    registered: number;
    owners: number;
    admins: number;
    moderators: number;
    djs: number;
    disabled: number;
  };
  streamers: {
    total: number;
    active: number;
    inactive: number;
    disabled: number;
  };
  schedule: {
    upcoming: number;
    live: number;
    reminders: number;
  };
  stars: {
    enabled: boolean;
    wallets: number;
    pendingOrders: number;
    completedOrders: number;
    sendEvents: number;
  };
};

function CommandCenter({ summary }: { summary?: CommandCenterSummary }) {
  if (!summary) {
    return null;
  }

  return (
    <div className="bouncecast-studio-dashboard">
      <div className="studio-hero">
        <div>
          <span className="studio-eyebrow">Command center</span>
          <h1>BounceCast operations</h1>
          <p>One snapshot for accounts, DJ access, public schedule, and Stars activity.</p>
        </div>
      </div>
      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic
              title="Registered accounts"
              value={summary.accounts.registered}
              suffix={`/ ${summary.accounts.total}`}
              prefix={<UserOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic
              title="Active DJs"
              value={summary.streamers.active}
              suffix={`/ ${summary.streamers.total}`}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic
              title="Upcoming sets"
              value={summary.schedule.upcoming}
              suffix={`${summary.schedule.reminders} reminders`}
              prefix={<CalendarOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic
              title="Stars sends"
              value={summary.stars.sendEvents}
              prefix={<StarOutlined />}
            />
          </Card>
        </Col>
      </Row>
      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={8}>
          <Card className="studio-panel">
            <Statistic
              title="Privileged account roles"
              value={
                summary.accounts.owners + summary.accounts.admins + summary.accounts.moderators
              }
              suffix={`roles, ${summary.accounts.djs} DJs`}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card className="studio-panel">
            <Statistic
              title="Pending DJ approvals"
              value={summary.streamers.inactive}
              suffix={`${summary.streamers.disabled} disabled`}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card className="studio-panel">
            <Statistic
              title="Stars wallets"
              value={summary.stars.wallets}
              suffix={summary.stars.enabled ? 'enabled' : 'disabled'}
              prefix={<StarOutlined />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}

function streamDetailsFormatter(streamDetails) {
  return (
    <ul className="statistics-list">
      <li>
        {streamDetails.videoCodec || 'Unknown'} @ {streamDetails.videoBitrate || 'Unknown'} kbps
      </li>
      <li>{streamDetails.framerate || 'Unknown'} fps</li>
      <li>
        {streamDetails.width} x {streamDetails.height}
      </li>
    </ul>
  );
}

export default function Home() {
  const { t } = useTranslation();

  const serverStatusData = useContext(ServerStatusContext);
  const { broadcaster, serverConfig: configData } = serverStatusData || {};
  const { remoteAddr, streamDetails } = broadcaster || {};

  const encoder = streamDetails?.encoder || 'Unknown encoder';

  const [logsData, setLogs] = useState([]);
  const [commandCenter, setCommandCenter] = useState<CommandCenterSummary | undefined>();
  const getLogs = async () => {
    try {
      const result = await fetchData(LOGS_WARN);
      setLogs(result);
    } catch (error) {
      console.log('==== error', error);
    }
  };
  const getMoreStats = () => {
    getLogs();
    fetchData(BOUNCECAST_COMMAND_CENTER)
      .then(result => setCommandCenter(result))
      .catch(error => console.log('==== command center error', error));
  };

  useEffect(() => {
    getMoreStats();

    let intervalId = null;
    intervalId = setInterval(getMoreStats, FETCH_INTERVAL);

    return () => {
      clearInterval(intervalId);
    };
  }, []);

  if (isEmptyObject(configData) || isEmptyObject(serverStatusData)) {
    return (
      <>
        <Skeleton active />
        <Skeleton active />
        <Skeleton active />
      </>
    );
  }

  if (!broadcaster) {
    return (
      <>
        <CommandCenter summary={commandCenter} />
        <Offline logs={logsData} config={configData} />
      </>
    );
  }

  // map out settings
  const videoQualitySettings = serverStatusData?.currentBroadcast?.outputSettings?.map(setting => {
    const { audioPassthrough, videoPassthrough, audioBitrate, videoBitrate, framerate } = setting;

    const audioSetting = audioPassthrough
      ? `${streamDetails.audioCodec || 'Unknown'}, ${streamDetails.audioBitrate} kbps`
      : `${audioBitrate || 'Unknown'} kbps`;

    const videoSetting = videoPassthrough
      ? `${streamDetails.videoBitrate || 'Unknown'} kbps, ${streamDetails.framerate} fps ${
          streamDetails.width
        } x ${streamDetails.height}`
      : `${videoBitrate || 'Unknown'} kbps, ${framerate} fps`;

    return (
      <div className="stream-details-item-container">
        <Statistic
          className="stream-details-item"
          title={t('Outbound Video Stream')}
          value={videoSetting}
        />
        <Statistic
          className="stream-details-item"
          title={t('Outbound Audio Stream')}
          value={audioSetting}
        />
      </div>
    );
  });

  // inbound
  const { viewerCount, sessionPeakViewerCount } = serverStatusData;

  const streamAudioDetailString = `${streamDetails.audioCodec}, ${
    streamDetails.audioBitrate || 'Unknown'
  } kbps`;

  const broadcastDate = new Date(broadcaster.time);

  return (
    <div className="home-container">
      <CommandCenter summary={commandCenter} />
      <div className="sections-container">
        <div className="online-status-section">
          <Card size="small" type="inner" className="online-details-card">
            <Row gutter={[16, 16]} align="middle">
              <Col span={8} sm={24} md={8}>
                <Statistic
                  title={`${t('Stream started')} ${formatRelative(broadcastDate, Date.now())}`}
                  value={formatDistanceToNow(broadcastDate)}
                  prefix={<ClockCircleOutlined />}
                />
              </Col>
              <Col span={8} sm={24} md={8}>
                <Statistic title={t('Viewers')} value={viewerCount} prefix={<UserOutlined />} />
              </Col>
              <Col span={8} sm={24} md={8}>
                <Statistic
                  title={t('Peak viewer count')}
                  value={sessionPeakViewerCount}
                  prefix={<UserOutlined />}
                />
              </Col>
            </Row>
            <StreamHealthOverview />
          </Card>
        </div>

        <Row gutter={[16, 16]} className="section stream-details-section">
          <Col className="stream-details" span={12} sm={24} md={24} lg={12}>
            <Card
              size="small"
              title={t('Outbound Stream Details')}
              type="inner"
              className="outbound-details"
            >
              {videoQualitySettings}
            </Card>

            <Card size="small" title={t('Inbound Stream Details')} type="inner">
              <Statistic
                className="stream-details-item"
                title={t('Input')}
                value={`${encoder} ${formatIPAddress(remoteAddr)}`}
              />
              <Statistic
                className="stream-details-item"
                title={t('Inbound Video Stream')}
                value={streamDetails}
                formatter={streamDetailsFormatter}
              />
              <Statistic
                className="stream-details-item"
                title={t('Inbound Audio Stream')}
                value={streamAudioDetailString}
              />
            </Card>
          </Col>

          <Col span={12} xs={24} sm={24} md={24} lg={12}>
            <NewsFeed />
          </Col>
        </Row>
      </div>
      <br />
      <LogTable logs={logsData} initialPageSize={5} />
    </div>
  );
}

Home.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
