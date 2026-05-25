import { Alert, Button, Col, Collapse, Input, Row, Space, Typography, message } from 'antd';
import React, { ReactElement, useEffect, useMemo, useState } from 'react';
import { CodecSelector as VideoCodecSelector } from '../../components/admin/CodecSelector';
import { VideoLatency } from '../../components/admin/VideoLatency';
import { CurrentVariantsTable } from '../../components/admin/CurrentVariantsTable';

import { AdminLayout } from '../../components/layouts/AdminLayout';

const { Panel } = Collapse;
const { Paragraph, Text, Title } = Typography;

export default function ConfigVideoSettings() {
  const [publicOrigin, setPublicOrigin] = useState('');

  useEffect(() => {
    setPublicOrigin(window.location.origin);
  }, []);

  const embedURL = `${publicOrigin || ''}/embed/video?autoplay=1&muted=0`;
  const embedCode = useMemo(
    () =>
      [
        '<iframe',
        `  src="${embedURL}"`,
        '  title="BounceCast live stream"',
        '  width="100%"',
        '  height="480"',
        '  style="border:0;aspect-ratio:16/9;width:100%;height:auto;"',
        '  allow="autoplay; fullscreen; picture-in-picture"',
        '  referrerpolicy="strict-origin-when-cross-origin">',
        '</iframe>',
      ].join('\n'),
    [embedURL],
  );

  const copyEmbedCode = async () => {
    try {
      await navigator.clipboard.writeText(embedCode);
      message.success('Video embed code copied');
    } catch {
      message.error('Could not copy the embed code automatically');
    }
  };

  return (
    <div className="config-video-variants">
      <Title>Video configuration</Title>
      <p className="description">
        Before changing your video configuration{' '}
        <a
          href="https://owncast.online/docs/video?source=admin"
          target="_blank"
          rel="noopener noreferrer"
        >
          visit the video documentation
        </a>{' '}
        to learn how it impacts your stream performance. The general rule is to start conservatively
        by having one middle quality stream output variant and experiment with adding more of varied
        qualities.
      </p>

      <Row gutter={[45, 16]}>
        <Col span={24}>
          <div className="form-module embed-code-module">
            <Title level={2}>Video embed</Title>
            <Paragraph>
              Add this iframe to another website to embed the live player. The embed requests
              autoplay with sound and includes the browser permission needed for autoplay.
            </Paragraph>
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              <div>
                <Text strong>Embed URL</Text>
                <Input value={embedURL} readOnly />
              </div>
              <div>
                <Text strong>Iframe code</Text>
                <Input.TextArea value={embedCode} readOnly rows={9} />
              </div>
              <Space wrap>
                <Button type="primary" onClick={copyEmbedCode}>
                  Copy embed code
                </Button>
                <Button href={embedURL} target="_blank" rel="noopener noreferrer">
                  Preview embed
                </Button>
              </Space>
              <Alert
                type="warning"
                showIcon
                message="Audible autoplay depends on the viewer's browser"
                description="Chrome, Safari, Firefox, and mobile browsers can block autoplay with sound until a visitor interacts with the page. This embed asks for sound; if the browser blocks it, the player will still be available for the visitor to start manually."
              />
            </Space>
          </div>
        </Col>
        <Col md={24} lg={12}>
          <div className="form-module variants-table-module">
            <CurrentVariantsTable />
          </div>
        </Col>
        <Col md={24} lg={12}>
          <div className="form-module latency-module">
            <VideoLatency />
          </div>

          <Collapse className="advanced-settings codec-module">
            <Panel header="Advanced Settings" key="1">
              <div className="form-module variants-table-module">
                <VideoCodecSelector />
              </div>
            </Panel>
          </Collapse>
        </Col>
      </Row>
    </div>
  );
}

ConfigVideoSettings.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
