import React, { ReactElement, useEffect, useMemo, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  Avatar,
  Button,
  Card,
  Checkbox,
  Col,
  Input,
  Row,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import { AdminLayout } from '../../components/layouts/AdminLayout';
import {
  BOUNCECAST_USER_PERMISSIONS,
  BOUNCECAST_USERS,
  USER_ENABLED_TOGGLE,
  fetchData,
} from '../../utils/apis';

const SaveOutlined = dynamic(() => import('@ant-design/icons/SaveOutlined'), { ssr: false });
const StopOutlined = dynamic(() => import('@ant-design/icons/StopOutlined'), { ssr: false });
const CheckCircleOutlined = dynamic(() => import('@ant-design/icons/CheckCircleOutlined'), {
  ssr: false,
});

const { Title, Text } = Typography;

type AccountUser = {
  id: string;
  displayName: string;
  email: string;
  profileImageUrl: string;
  permissions: string[];
  scopes: string[];
  displayColor: number;
  registered: boolean;
  authenticated: boolean;
  messageCount: number;
  createdAt: string;
  disabledAt?: string;
  registeredAt?: string;
  lastLoginAt?: string;
};

const permissionOptions = [
  { label: 'Owner', value: 'owner' },
  { label: 'Admin', value: 'admin' },
  { label: 'Moderator', value: 'moderator' },
  { label: 'DJ', value: 'dj' },
];

const permissionColors = {
  owner: 'gold',
  admin: 'magenta',
  moderator: 'cyan',
  dj: 'blue',
  visitor: 'default',
};

function editablePermissions(permissions: string[]) {
  return (permissions || []).filter(permission => permission !== 'visitor');
}

function displayPermissions(permissions: string[]) {
  const roles = permissions?.length ? permissions : ['visitor'];
  return roles.map(permission => (
    <Tag key={permission} color={permissionColors[permission] || 'default'}>
      {permission}
    </Tag>
  ));
}

function formatDate(value?: string) {
  return value ? new Date(value).toLocaleString() : 'Never';
}

export default function AccountsAdmin() {
  const [users, setUsers] = useState<AccountUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingUserId, setSavingUserId] = useState('');
  const [search, setSearch] = useState('');
  const [pendingPermissions, setPendingPermissions] = useState<Record<string, string[]>>({});

  const loadUsers = async () => {
    setLoading(true);
    try {
      const result = await fetchData(BOUNCECAST_USERS);
      setUsers(result || []);
      setPendingPermissions({});
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to load accounts');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadUsers();
  }, []);

  const filteredUsers = useMemo(() => {
    const needle = search.trim().toLowerCase();
    if (!needle) {
      return users;
    }
    return users.filter(user =>
      [user.displayName, user.email, user.id].some(value =>
        (value || '').toLowerCase().includes(needle),
      ),
    );
  }, [search, users]);

  const savePermissions = async (user: AccountUser) => {
    setSavingUserId(user.id);
    try {
      await fetchData(BOUNCECAST_USER_PERMISSIONS, {
        method: 'POST',
        data: {
          userId: user.id,
          permissions: pendingPermissions[user.id] || editablePermissions(user.permissions),
        },
      });
      message.success('Permissions saved');
      await loadUsers();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to save permissions');
    } finally {
      setSavingUserId('');
    }
  };

  const setEnabled = async (user: AccountUser, enabled: boolean) => {
    setSavingUserId(user.id);
    try {
      await fetchData(USER_ENABLED_TOGGLE, {
        method: 'POST',
        data: { userId: user.id, enabled },
      });
      message.success(enabled ? 'User enabled' : 'User disabled');
      await loadUsers();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Unable to update user');
    } finally {
      setSavingUserId('');
    }
  };

  const registeredCount = users.filter(user => user.registered).length;
  const djCount = users.filter(user => user.permissions.includes('dj')).length;
  const moderatorCount = users.filter(user => user.permissions.includes('moderator')).length;

  const columns = [
    {
      title: 'User',
      key: 'user',
      render: (_, record: AccountUser) => (
        <Space>
          <Avatar src={record.profileImageUrl}>{record.displayName?.slice(0, 1)}</Avatar>
          <Space direction="vertical" size={0}>
            <Text strong>{record.displayName}</Text>
            <Text type="secondary">{record.email || record.id}</Text>
          </Space>
        </Space>
      ),
    },
    {
      title: 'Current',
      key: 'current',
      render: (_, record: AccountUser) => displayPermissions(record.permissions),
    },
    ...permissionOptions.map(option => ({
      title: option.label,
      key: option.value,
      align: 'center' as const,
      render: (_, record: AccountUser) => {
        const activePermissions =
          pendingPermissions[record.id] || editablePermissions(record.permissions);
        return (
          <Checkbox
            checked={activePermissions.includes(option.value)}
            onChange={event => {
              const checked = event.target.checked;
              const nextPermissions = checked
                ? [...activePermissions, option.value]
                : activePermissions.filter(permission => permission !== option.value);
              setPendingPermissions(current => ({
                ...current,
                [record.id]: nextPermissions,
              }));
            }}
            aria-label={`${option.label} permission for ${record.displayName}`}
          />
        );
      },
    })),
    {
      title: 'Status',
      key: 'status',
      render: (_, record: AccountUser) => (
        <Space direction="vertical" size={0}>
          <Tag color={record.disabledAt ? 'default' : 'green'}>
            {record.disabledAt ? 'disabled' : 'enabled'}
          </Tag>
          <Tag color={record.registered ? 'cyan' : 'default'}>
            {record.registered ? 'registered' : 'visitor'}
          </Tag>
        </Space>
      ),
    },
    {
      title: 'Last login',
      dataIndex: 'lastLoginAt',
      key: 'lastLoginAt',
      render: formatDate,
    },
    {
      title: 'Chat',
      dataIndex: 'messageCount',
      key: 'messageCount',
      render: count => <Tag color={count > 0 ? 'blue' : 'default'}>{count}</Tag>,
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record: AccountUser) => (
        <Space>
          <Button
            size="small"
            type="primary"
            icon={<SaveOutlined />}
            loading={savingUserId === record.id}
            onClick={() => savePermissions(record)}
          >
            Save
          </Button>
          {record.disabledAt ? (
            <Button
              size="small"
              icon={<CheckCircleOutlined />}
              loading={savingUserId === record.id}
              onClick={() => setEnabled(record, true)}
            >
              Enable
            </Button>
          ) : (
            <Button
              size="small"
              danger
              icon={<StopOutlined />}
              loading={savingUserId === record.id}
              onClick={() => setEnabled(record, false)}
            >
              Disable
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="bouncecast-studio-dashboard">
      <div className="studio-hero">
        <div>
          <Text className="studio-eyebrow">Account control</Text>
          <Title level={1}>Users & permissions</Title>
          <Text>
            Visitor is the default. Owner, admin, moderator, and DJ permissions can be assigned
            independently.
          </Text>
        </div>
      </div>

      <Row gutter={[16, 16]} className="studio-stat-row">
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic title="Users" value={users.length} />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic title="Registered" value={registeredCount} />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic title="Moderators" value={moderatorCount} />
          </Card>
        </Col>
        <Col xs={24} md={6}>
          <Card className="studio-panel">
            <Statistic title="DJs" value={djCount} />
          </Card>
        </Col>
      </Row>

      <Card
        className="studio-panel"
        title="Accounts"
        extra={
          <Input.Search
            allowClear
            placeholder="Search users"
            value={search}
            onChange={event => setSearch(event.target.value)}
            style={{ width: 260 }}
          />
        }
      >
        <Table
          loading={loading}
          columns={columns}
          dataSource={filteredUsers}
          rowKey="id"
          scroll={{ x: true }}
        />
      </Card>
    </div>
  );
}

AccountsAdmin.getLayout = function getLayout(page: ReactElement) {
  return <AdminLayout page={page} />;
};
