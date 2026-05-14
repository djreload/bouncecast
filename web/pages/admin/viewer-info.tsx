import React, { useState, useEffect, useContext, ReactElement } from 'react';
import { Row, Col, Typography, MenuProps, Dropdown, Spin, Alert, Card } from 'antd';
import { getUnixTime, sub } from 'date-fns';
import dynamic from 'next/dynamic';
import { useTranslation } from 'next-export-i18n';
import { Chart } from '../../components/admin/Chart';
import { StatisticItem } from '../../components/admin/StatisticItem';
import { ViewerTable } from '../../components/admin/ViewerTable';

import { ServerStatusContext } from '../../utils/server-status-context';

import { VIEWERS_OVER_TIME, ACTIVE_VIEWER_DETAILS, fetchData } from '../../utils/apis';

import { AdminLayout } from '../../components/layouts/AdminLayout';
import { Localization } from '../../types/localization';
import styles from './viewer-info.module.scss';

// Lazy loaded components

const DownOutlined = dynamic(() => import('@ant-design/icons/DownOutlined'), {
  ssr: false,
});

const UserOutlined = dynamic(() => import('@ant-design/icons/UserOutlined'), {
  ssr: false,
});

const FETCH_INTERVAL = 60 * 1000; // 1 min

export default function ViewersOverTime() {
  const context = useContext(ServerStatusContext);
  const { t } = useTranslation();
  const { online, broadcaster, viewerCount, overallPeakViewerCount, sessionPeakViewerCount } =
    context || {};
  let streamStart;
  if (broadcaster && broadcaster.time) {
    streamStart = new Date(broadcaster.time);
  }

  const times = [
    { title: t(Localization.Admin.ViewerInfo.currentStream), start: streamStart },
    { title: t(Localization.Admin.ViewerInfo.last12Hours), start: sub(new Date(), { hours: 12 }) },
    { title: t(Localization.Admin.ViewerInfo.last24Hours), start: sub(new Date(), { hours: 24 }) },
    { title: t(Localization.Admin.ViewerInfo.last7Days), start: sub(new Date(), { days: 7 }) },
    { title: t(Localization.Admin.ViewerInfo.last30Days), start: sub(new Date(), { days: 30 }) },
    { title: t(Localization.Admin.ViewerInfo.last3Months), start: sub(new Date(), { months: 3 }) },
    { title: t(Localization.Admin.ViewerInfo.last6Months), start: sub(new Date(), { months: 6 }) },
  ];

  const [loadingChart, setLoadingChart] = useState(true);
  const [viewerInfo, setViewerInfo] = useState([]);
  const [viewerDetails, setViewerDetails] = useState([]);
  const [timeWindowStart, setTimeWindowStart] = useState(times[1]);
  const [timeWindowKey, setTimeWindowKey] = useState(1);

  const getInfo = async () => {
    try {
      const url = `${VIEWERS_OVER_TIME}?windowStart=${getUnixTime(timeWindowStart.start)}`;
      const result = await fetchData(url);
      setViewerInfo(result);
      setLoadingChart(false);
    } catch (error) {
      console.log('==== error', error);
    }

    try {
      const result = await fetchData(ACTIVE_VIEWER_DETAILS);
      setViewerDetails(result);
    } catch (error) {
      console.log('==== error', error);
    }
  };

  useEffect(() => {
    let getStatusIntervalId = null;

    getInfo();
    if (online) {
      getStatusIntervalId = setInterval(getInfo, FETCH_INTERVAL);
      // returned function will be called on component unmount
      return () => {
        clearInterval(getStatusIntervalId);
      };
    }

    return () => [];
  }, [online, timeWindowStart]);

  const onTimeWindowSelect = ({ key }) => {
    setTimeWindowStart(times[key]);
    setTimeWindowKey(Number(key));
  };

  const offset: number = online && streamStart ? 0 : 1;
  const items: MenuProps['items'] = times.slice(offset).map((time, index) => ({
    key: index + offset,
    label: time.title,
    onClick: onTimeWindowSelect,
  }));

  return (
    <div className={`bouncecast-admin-page viewer-info-page ${styles.viewerInfoPage}`}>
      <div className="studio-hero">
        <div>
          <Typography.Text className="studio-eyebrow">Audience signal</Typography.Text>
          <Typography.Title level={1}>{t(Localization.Admin.ViewerInfo.title)}</Typography.Title>
          <Typography.Text>
            Live viewer activity, audience peaks, and connection details.
          </Typography.Text>
        </div>
      </div>

      <Row gutter={[16, 16]} justify="space-around" className="studio-stat-row viewer-stat-row">
        {online && (
          <Col span={8} md={8}>
            <StatisticItem
              title={t(Localization.Admin.ViewerInfo.currentViewers)}
              value={viewerCount.toString()}
              prefix={<UserOutlined />}
            />
          </Col>
        )}
        <Col md={online ? 8 : 12}>
          <StatisticItem
            title={
              online
                ? t(Localization.Admin.ViewerInfo.maxViewersThisStream)
                : t(Localization.Admin.ViewerInfo.maxViewersLastStream)
            }
            value={sessionPeakViewerCount.toString()}
            prefix={<UserOutlined />}
          />
        </Col>
        <Col md={online ? 8 : 12}>
          <StatisticItem
            title={t(Localization.Admin.ViewerInfo.maxViewers)}
            value={overallPeakViewerCount.toString()}
            prefix={<UserOutlined />}
          />
        </Col>
      </Row>
      {!viewerInfo.length && (
        <Alert
          style={{ marginTop: '10px' }}
          banner
          message={t(Localization.Admin.ViewerInfo.pleaseWait)}
          description={t(Localization.Admin.ViewerInfo.noData)}
          type="info"
        />
      )}

      <Card className="studio-panel viewer-chart-panel">
        {viewerInfo.length > 0 && (
          <Spin spinning={loadingChart}>
            <Chart
              title={t(Localization.Admin.ViewerInfo.viewers)}
              data={viewerInfo}
              color="#20e4ff"
              unit="viewers"
              minYValue={0}
              yStepSize={1}
              timeWindowKey={timeWindowKey}
            />
          </Spin>
        )}

        <Dropdown menu={{ items }} trigger={['click']}>
          <button type="button" className="viewer-window-button">
            {timeWindowStart.title} <DownOutlined />
          </button>
        </Dropdown>
        <ViewerTable data={viewerDetails} />
      </Card>
    </div>
  );
}

ViewersOverTime.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
