import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Form, Card } from 'semantic-ui-react';
import { API, showError, showSuccess } from '../../helpers';

const AddUser = () => {
  const { t } = useTranslation();
  const originInputs = {
    username: '',
    display_name: '',
    password: '',
  };
  const [inputs, setInputs] = useState(originInputs);
  const { username, display_name, password } = inputs;

  const handleInputChange = (e, { name, value }) => {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const submit = async () => {
    // 前端验证
    if (inputs.username === '') {
      showError('用户名不能为空');
      return;
    }
    if (inputs.password === '') {
      showError('密码不能为空');
      return;
    }
    if (inputs.username.length > 12) {
      showError('用户名长度不能超过12个字符');
      return;
    }
    if (inputs.password.length < 8) {
      showError('密码长度不能少于8个字符');
      return;
    }
    if (inputs.password.length > 20) {
      showError('密码长度不能超过20个字符');
      return;
    }
    if (inputs.display_name && inputs.display_name.length > 20) {
      showError('显示名称长度不能超过20个字符');
      return;
    }

    const res = await API.post(`/api/user/`, inputs);
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('user.messages.create_success'));
      setInputs(originInputs);
    } else {
      showError(message);
    }
  };

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header className='header'>{t('user.add.title')}</Card.Header>
          <Form autoComplete='off'>
            <Form.Field>
              <Form.Input
                label={t('user.edit.username')}
                name='username'
                placeholder={t('user.edit.username_placeholder')}
                onChange={handleInputChange}
                value={username}
                autoComplete='off'
                required
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('user.edit.display_name')}
                name='display_name'
                placeholder={t('user.edit.display_name_placeholder')}
                onChange={handleInputChange}
                value={display_name}
                autoComplete='off'
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('user.edit.password')}
                name='password'
                type='password'
                placeholder={t('user.edit.password_placeholder')}
                onChange={handleInputChange}
                value={password}
                autoComplete='off'
                required
              />
            </Form.Field>
            <Button positive type='submit' onClick={submit}>
              {t('user.edit.buttons.submit')}
            </Button>
          </Form>
        </Card.Content>
      </Card>
    </div>
  );
};

export default AddUser;
