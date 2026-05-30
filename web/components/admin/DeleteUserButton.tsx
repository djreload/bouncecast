import { Button, Modal, message } from 'antd';
import dynamic from 'next/dynamic';
import { FC } from 'react';
import { USER_DELETE, fetchData } from '../../utils/apis';
import { User } from '../../types/chat';

const DeleteOutlined = dynamic(() => import('@ant-design/icons/DeleteOutlined'), { ssr: false });

export type DeleteUserButtonProps = {
  user: User;
  label?: string;
  onDeleted?: () => void;
};

export const DeleteUserButton: FC<DeleteUserButtonProps> = ({ user, label, onDeleted }) => {
  const confirmDelete = () => {
    Modal.confirm({
      title: 'Delete user',
      content: (
        <>
          Permanently delete <strong>{user.displayName}</strong> and owned records for user ID{' '}
          <code>{user.id}</code>?
        </>
      ),
      okText: 'Delete',
      okType: 'danger',
      onOk: async () => {
        const result = await fetchData(USER_DELETE, {
          method: 'POST',
          data: { userId: user.id },
        });
        if (result?.success === false) {
          throw new Error(result.message || 'Unable to delete user');
        }
        message.success('User deleted');
        onDeleted?.();
      },
    });
  };

  return (
    <Button danger size="small" icon={<DeleteOutlined />} onClick={confirmDelete}>
      {label || 'Delete'}
    </Button>
  );
};

DeleteUserButton.defaultProps = {
  label: '',
  onDeleted: undefined,
};
