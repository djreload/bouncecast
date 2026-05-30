import { Space, Table, Typography } from 'antd';
import { format } from 'date-fns';
import { SortOrder } from 'antd/lib/table/interface';
import { FC } from 'react';
import { User } from '../../types/chat';
import { UserPopover } from './UserPopover';
import { BanUserButton } from './BanUserButton';
import { DeleteUserButton } from './DeleteUserButton';

export function formatDisplayDate(date: string | Date) {
  const d = new Date(date);
  if (d.getFullYear() !== new Date().getFullYear()) {
    return format(new Date(date), 'MMM d, yyyy H:mma');
  }

  return format(new Date(date), 'MMM d H:mma');
}

export type UserTableProps = {
  data: User[];
  onRefresh?: () => void;
};

export const UserTable: FC<UserTableProps> = ({ data, onRefresh }) => {
  const columns = [
    {
      title: 'Last Known Display Name',
      dataIndex: 'displayName',
      key: 'displayName',
      // eslint-disable-next-line react/destructuring-assignment
      render: (displayName: string, user: User) => (
        <UserPopover user={user} onUserChanged={onRefresh}>
          <Space direction="vertical" size={0}>
            <span className="display-name">{displayName}</span>
            <Typography.Text type="secondary" copyable>
              {user.id}
            </Typography.Text>
          </Space>
        </UserPopover>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (date: Date) => formatDisplayDate(date),
      sorter: (a: any, b: any) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      sortDirections: ['descend', 'ascend'] as SortOrder[],
    },
    {
      title: 'Disabled at',
      dataIndex: 'disabledAt',
      key: 'disabledAt',
      defaultSortOrder: 'descend' as SortOrder,
      render: (date: Date) => (date ? formatDisplayDate(date) : null),
      sorter: (a: any, b: any) =>
        new Date(a.disabledAt).getTime() - new Date(b.disabledAt).getTime(),
      sortDirections: ['descend', 'ascend'] as SortOrder[],
    },
    {
      title: '',
      key: 'block',
      className: 'actions-col',
      render: (_, user) => (
        <Space>
          <BanUserButton user={user} isEnabled={!user.disabledAt} onClick={onRefresh} />
          <DeleteUserButton user={user} onDeleted={onRefresh} />
        </Space>
      ),
    },
  ];

  return (
    <Table
      pagination={{ hideOnSinglePage: true }}
      className="table-container"
      columns={columns}
      dataSource={data}
      size="small"
      rowKey="id"
    />
  );
};

UserTable.defaultProps = {
  onRefresh: null,
};
