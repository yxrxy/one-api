import React, { useState, useEffect } from 'react';
import {
  Button,
  Card,
  Table,
  Label,
  Modal,
  Form,
  Input,
  Dropdown,
  Message,
  Grid,
  Header,
  Icon,
  Segment,
  Statistic,
  Divider
} from 'semantic-ui-react';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';

const Payment = () => {
  const { t } = useTranslation();
  const [payments, setPayments] = useState([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [formData, setFormData] = useState({
    amount: '',
    payment_method: '',
    top_up_code: ''
  });

  const paymentMethodOptions = [
    { key: 'alipay', text: '支付宝', value: 'alipay' },
    { key: 'wechat', text: '微信支付', value: 'wechat' },
    { key: 'paypal', text: 'PayPal', value: 'paypal' }
  ];

  const statusMap = {
    'pending': { text: '待处理', color: 'orange' },
    'processing': { text: '处理中', color: 'blue' },
    'success': { text: '支付成功', color: 'green' },
    'failed': { text: '支付失败', color: 'red' }
  };

  const fetchPayments = async (page = 1, size = 10) => {
    setLoading(true);
    try {
      const response = await API.get(`/api/user/payments?page=${page}&page_size=${size}`);
      if (response.data.success) {
        setPayments(response.data.data);
        setTotal(response.data.total);
      } else {
        showError(response.data.message);
      }
    } catch (error) {
      showError('获取支付记录失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPayments();
  }, []);

  const handlePageChange = (page, size) => {
    setCurrentPage(page);
    setPageSize(size);
    fetchPayments(page, size);
  };

  const handleCreatePayment = async () => {
    if (!formData.amount || !formData.payment_method) {
      showError('请填写完整信息');
      return;
    }

    try {
      // 确保 amount 是数字类型
      const paymentData = {
        ...formData,
        amount: parseFloat(formData.amount)
      };
      const response = await API.post('/api/user/pay', paymentData);
      if (response.data.success) {
        showSuccess('创建支付订单成功');
        setIsModalOpen(false);
        setFormData({ amount: '', payment_method: '', top_up_code: '' });
        fetchPayments();
        
        // 如果是支付宝，直接跳转
        if (formData.payment_method === 'alipay') {
          window.open(response.data.url, '_blank');
        }
        // 如果是微信支付，显示二维码
        else if (formData.payment_method === 'wechat') {
          Modal.alert({
            title: '微信支付',
            content: (
              <div style={{ textAlign: 'center' }}>
                <p>请使用微信扫描以下二维码完成支付：</p>
                <img src={response.data.data.code_url} alt="支付二维码" style={{ width: '200px' }} />
              </div>
            )
          });
        }
      } else {
        showError(response.data.message);
      }
    } catch (error) {
      showError('创建支付订单失败');
    }
  };

  const handleInputChange = (e, { name, value }) => {
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  const handleContinuePayment = async (record) => {
    if (record.payment_method === 'alipay' && record.status === 'pending') {
      try {
        const response = await API.post('/api/user/pay', {
          amount: parseFloat(record.amount) / 1000, // 将额度转换为金额
          payment_method: record.payment_method,
          top_up_code: record.top_up_code,
        });
        if (response.data.success) {
          window.open(response.data.url, '_blank');
        }
      } catch (error) {
        showError('获取支付链接失败');
      }
    }
  };

  const showPaymentDetails = (record) => {
    Modal.alert({
      title: '订单详情',
      content: (
        <div>
          <p><strong>订单号：</strong>{record.order_id}</p>
          <p><strong>充值数量：</strong>{record.amount} 额度</p>
          <p><strong>支付方式：</strong>{record.payment_method}</p>
          <p><strong>状态：</strong>{statusMap[record.status]?.text || record.status}</p>
          <p><strong>创建时间：</strong>{new Date(record.created_at).toLocaleString()}</p>
          {record.top_up_code && (
            <p><strong>充值码：</strong>{record.top_up_code}</p>
          )}
        </div>
      )
    });
  };

  const tableHeaders = [
    { key: 'order_id', text: '订单号', width: 200 },
    { key: 'amount', text: '充值数量', width: 120 },
    { key: 'payment_method', text: '支付方式', width: 120 },
    { key: 'status', text: '状态', width: 100 },
    { key: 'created_at', text: '创建时间', width: 150 },
    { key: 'action', text: '操作', width: 150 }
  ];

  return (
    <div className='dashboard-container'>
      <Card fluid className='chart-card'>
        <Card.Content>
          <Card.Header>
            <Header as='h2'>
              <Icon name='credit card' />
              <Header.Content>
                支付记录
                <Header.Subheader>管理您的支付订单</Header.Subheader>
              </Header.Content>
            </Header>
          </Card.Header>

          <Grid columns={2} stackable>
            <Grid.Column>
              <Card fluid>
                <Card.Content>
                  <Card.Header>支付统计</Card.Header>
                  <Card.Description>
                    <Statistic.Group size='small'>
                      <Statistic>
                        <Statistic.Value>{total}</Statistic.Value>
                        <Statistic.Label>总订单数</Statistic.Label>
                      </Statistic>
                      <Statistic>
                        <Statistic.Value>
                          {payments.filter(p => p.status === 'success').length}
                        </Statistic.Value>
                        <Statistic.Label>成功订单</Statistic.Label>
                      </Statistic>
                    </Statistic.Group>
                  </Card.Description>
                </Card.Content>
              </Card>
            </Grid.Column>
            <Grid.Column>
              <Button 
                primary 
                fluid 
                icon 
                labelPosition='left'
                onClick={() => setIsModalOpen(true)}
              >
                <Icon name='plus' />
                创建支付订单
              </Button>
            </Grid.Column>
          </Grid>

          <Divider />

          <Table celled selectable>
            <Table.Header>
              <Table.Row>
                {tableHeaders.map(header => (
                  <Table.HeaderCell key={header.key} width={header.width}>
                    {header.text}
                  </Table.HeaderCell>
                ))}
              </Table.Row>
            </Table.Header>

            <Table.Body>
              {payments.map(payment => (
                <Table.Row key={payment.id}>
                  <Table.Cell>{payment.order_id}</Table.Cell>
                  <Table.Cell>{payment.amount} 额度</Table.Cell>
                  <Table.Cell>
                    <Label color={
                      payment.payment_method === 'alipay' ? 'blue' :
                      payment.payment_method === 'wechat' ? 'green' :
                      payment.payment_method === 'paypal' ? 'orange' : 'grey'
                    }>
                      {payment.payment_method === 'alipay' ? '支付宝' :
                       payment.payment_method === 'wechat' ? '微信支付' :
                       payment.payment_method === 'paypal' ? 'PayPal' : payment.payment_method}
                    </Label>
                  </Table.Cell>
                  <Table.Cell>
                    <Label color={statusMap[payment.status]?.color || 'grey'}>
                      {statusMap[payment.status]?.text || payment.status}
                    </Label>
                  </Table.Cell>
                  <Table.Cell>{new Date(payment.created_at).toLocaleString()}</Table.Cell>
                  <Table.Cell>
                    <Button.Group size='mini'>
                      {(payment.status === 'pending' || payment.status === 'processing') && (
                        <Button 
                          primary
                          disabled={payment.status === 'processing'}
                          onClick={() => handleContinuePayment(payment)}
                        >
                          {payment.status === 'processing' ? '处理中' : '继续支付'}
                        </Button>
                      )}
                      <Button 
                        icon
                        onClick={() => showPaymentDetails(payment)}
                      >
                        <Icon name='info circle' />
                      </Button>
                    </Button.Group>
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table>

          {payments.length === 0 && !loading && (
            <Segment placeholder>
              <Header icon>
                <Icon name='credit card outline' />
                暂无支付记录
              </Header>
            </Segment>
          )}
        </Card.Content>
      </Card>

      <Modal
        open={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        size='small'
      >
        <Modal.Header>创建支付订单</Modal.Header>
        <Modal.Content>
          <Form>
            <Form.Field>
              <label>充值金额（元）</label>
              <Input
                type='number'
                name='amount'
                value={formData.amount}
                onChange={handleInputChange}
                placeholder='请输入充值金额（元）'
                min={0.01}
                step={0.01}
              />
            </Form.Field>
            <Form.Field>
              <label>支付方式</label>
              <Dropdown
                placeholder='请选择支付方式'
                fluid
                selection
                options={paymentMethodOptions}
                name='payment_method'
                value={formData.payment_method}
                onChange={(e, { value }) => handleInputChange(e, { name: 'payment_method', value })}
              />
            </Form.Field>
            <Form.Field>
              <label>充值码（可选）</label>
              <Input
                name='top_up_code'
                value={formData.top_up_code}
                onChange={handleInputChange}
                placeholder='请输入充值码'
              />
            </Form.Field>
          </Form>
        </Modal.Content>
        <Modal.Actions>
          <Button onClick={() => setIsModalOpen(false)}>
            取消
          </Button>
          <Button primary onClick={handleCreatePayment}>
            创建订单
          </Button>
        </Modal.Actions>
      </Modal>
    </div>
  );
};

export default Payment; 