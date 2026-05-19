import React, { ReactElement, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/router';
import {
  Avatar,
  Button,
  Card,
  Empty,
  Form,
  Input,
  Select,
  Skeleton,
  Space,
  Tag,
  Typography,
} from 'antd';
import {
  BounceCastDJProfile,
  BounceCastPublicDJ,
  BounceCastScheduleItem,
  BounceCastService,
} from '../services/bouncecast-service';
import styles from '../styles/bouncecast-public.module.scss';

const { Text } = Typography;

function formatDate(value?: string) {
  if (!value) {
    return 'To be announced';
  }
  return new Date(value).toLocaleString([], {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function ScheduleList({ schedule }: { schedule: BounceCastScheduleItem[] }) {
  if (!schedule.length) {
    return <Empty description="No public sets scheduled yet" />;
  }

  return (
    <div className={styles.scheduleList}>
      {schedule.map(item => (
        <article className={styles.scheduleItem} key={item.id}>
          <p className={styles.scheduleTime}>{formatDate(item.startsAt)}</p>
          <p className={styles.scheduleTitle}>{item.title}</p>
          <Text className={styles.muted}>
            {item.streamer ? `with ${item.streamer}` : 'BounceCast lineup'}
          </Text>
        </article>
      ))}
    </div>
  );
}

function DJCard({ dj, active }: { dj: BounceCastPublicDJ; active: boolean }) {
  return (
    <Card className={styles.panel} bodyStyle={{ height: '100%' }}>
      <div className={styles.djCard}>
        <div className={styles.djCardHeader}>
          <Avatar src={dj.avatarUrl} size={54}>
            {dj.displayName?.slice(0, 1)}
          </Avatar>
          <div>
            <p className={styles.djName}>{dj.displayName}</p>
            <p className={styles.handle}>@{dj.handle}</p>
          </div>
        </div>
        <div className={styles.statRow}>
          <Tag color={dj.role === 'resident' ? 'magenta' : 'cyan'}>{dj.role}</Tag>
          <Tag color="blue">{dj.upcomingCount} upcoming</Tag>
          <Tag color="purple">{dj.totalLiveEvents} live sets</Tag>
        </div>
        {dj.genres?.length > 0 && (
          <div className={styles.statRow}>
            {dj.genres.slice(0, 4).map(genre => (
              <Tag key={genre} color="geekblue">
                {genre}
              </Tag>
            ))}
          </div>
        )}
        {dj.bio && <Text className={styles.muted}>{dj.bio}</Text>}
        <Text className={styles.muted}>
          {dj.upcomingSet
            ? `Next: ${dj.upcomingSet} at ${formatDate(dj.upcomingStarts)}`
            : 'No upcoming public set yet.'}
        </Text>
        <Link href={`/djs?handle=${encodeURIComponent(dj.handle)}`}>
          <Button type={active ? 'primary' : 'default'} block>
            View profile
          </Button>
        </Link>
      </div>
    </Card>
  );
}

export default function DJsPage() {
  const router = useRouter();
  const [djs, setDjs] = useState<BounceCastPublicDJ[]>([]);
  const [profile, setProfile] = useState<BounceCastDJProfile | null>(null);
  const [lineup, setLineup] = useState<BounceCastScheduleItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [statusFilter, setStatusFilter] = useState<'all' | 'planned' | 'live'>('all');
  const [searchFilter, setSearchFilter] = useState('');

  const selectedHandle = useMemo(() => {
    const queryHandle = router.query.handle;
    return Array.isArray(queryHandle) ? queryHandle[0] : queryHandle || djs[0]?.handle || '';
  }, [djs, router.query.handle]);

  useEffect(() => {
    BounceCastService.getDJs()
      .then(result => setDjs(result || []))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    BounceCastService.getSchedule({
      limit: 25,
      status: statusFilter,
      q: searchFilter,
    })
      .then(result => setLineup(result || []))
      .catch(() => setLineup([]));
  }, [searchFilter, statusFilter]);

  useEffect(() => {
    if (!selectedHandle) {
      setProfile(null);
      return;
    }
    BounceCastService.getDJProfile(selectedHandle)
      .then(result => setProfile(result))
      .catch(() => setProfile(null));
  }, [selectedHandle]);

  return (
    <main className={styles.page}>
      <div className={styles.shell}>
        <header className={styles.topbar}>
          <Link href="/" className={styles.brand}>
            <img src="/logo" alt="BounceCast" />
            <p className={styles.brandText}>BounceCast</p>
          </Link>
          <nav className={styles.nav} aria-label="Public navigation">
            <Link href="/">Live stream</Link>
            <Link href="/account">Account Hub</Link>
            <Link href="/login">Dashboard login</Link>
          </nav>
        </header>

        <section className={styles.hero}>
          <div>
            <p className={styles.eyebrow}>DJ roster</p>
            <h1 className={styles.headline}>Lineup and profiles</h1>
            <p className={styles.copy}>
              Browse active BounceCast DJs, upcoming live sets, and recent stage activity.
            </p>
          </div>
          <Link href="/">
            <Button type="primary">Back to stream</Button>
          </Link>
        </section>

        {loading ? (
          <Card className={styles.panel}>
            <Skeleton active />
          </Card>
        ) : (
          <div className={styles.twoColumn}>
            <section className={styles.grid} aria-label="DJ profiles">
              {djs.length ? (
                djs.map(dj => (
                  <DJCard key={dj.handle} dj={dj} active={dj.handle === selectedHandle} />
                ))
              ) : (
                <Card className={styles.panel}>
                  <Empty description="No active DJs yet" />
                </Card>
              )}
            </section>

            <aside>
              <Card className={styles.panel} title="Lineup filters">
                <Form layout="vertical">
                  <Form.Item label="Search">
                    <Input
                      allowClear
                      maxLength={80}
                      placeholder="DJ, set, or genre"
                      value={searchFilter}
                      onChange={event => setSearchFilter(event.target.value)}
                    />
                  </Form.Item>
                  <Form.Item label="Status">
                    <Select
                      value={statusFilter}
                      onChange={setStatusFilter}
                      options={[
                        { label: 'All active sets', value: 'all' },
                        { label: 'Planned', value: 'planned' },
                        { label: 'Live now', value: 'live' },
                      ]}
                    />
                  </Form.Item>
                </Form>
              </Card>

              <Card className={styles.panel} title="Public lineup">
                <ScheduleList schedule={lineup} />
              </Card>

              <Card
                className={styles.panel}
                title={profile ? `${profile.dj.displayName} schedule` : 'DJ schedule'}
              >
                {profile?.dj.heroImageUrl && (
                  <img
                    className={styles.heroImage}
                    src={profile.dj.heroImageUrl}
                    alt={`${profile.dj.displayName} hero`}
                  />
                )}
                {profile?.dj.bio && <p className={styles.profileBio}>{profile.dj.bio}</p>}
                {profile?.dj.genres?.length > 0 && (
                  <Space wrap className={styles.profileTags}>
                    {profile.dj.genres.map(genre => (
                      <Tag key={genre} color="geekblue">
                        {genre}
                      </Tag>
                    ))}
                  </Space>
                )}
                {profile?.dj.socialLinks?.length > 0 && (
                  <Space wrap className={styles.profileTags}>
                    {profile.dj.socialLinks.map(link => (
                      <a key={link.url} href={link.url} rel="noreferrer" target="_blank">
                        {link.label}
                      </a>
                    ))}
                  </Space>
                )}
                <ScheduleList schedule={profile?.schedule || []} />
              </Card>
            </aside>
          </div>
        )}
      </div>
    </main>
  );
}

DJsPage.getLayout = function getLayout(page: ReactElement) {
  return page;
};
