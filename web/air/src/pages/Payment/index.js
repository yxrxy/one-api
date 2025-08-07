import React, { useState, useEffect } from 'react';
import { Card, Table, Tag, Button, Modal, Form, Input, Select, message, Space, Typography } from 'antd';
import { API } from '../../helpers/api';
import { showError, showSuccess } from '../../helpers/common';

const { Title } = Typography;
const { Option } = Select;

const Payment = () => {
    const [payments, setPayments] = useState([]);
    const [loading, setLoading] = useState(false);
    const [total, setTotal] = useState(0);
    const [currentPage, setCurrentPage] = useState(1);
    const [pageSize, setPageSize] = useState(10);
    const [isModalVisible, setIsModalVisible] = useState(false);
    const [form] = Form.useForm();

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

    const handleCreatePayment = async (values) => {
        try {
            const response = await API.post('/api/user/pay', values);
            if (response.data.success) {
                showSuccess('创建支付订单成功');
                setIsModalVisible(false);
                form.resetFields();
                fetchPayments();
                
                // 如果是支付宝，直接跳转
                if (values.payment_method === 'alipay') {
                    window.open(response.data.url, '_blank');
                }
                // 如果是微信支付，显示二维码
                else if (values.payment_method === 'wechat') {
                    Modal.info({
                        title: '微信支付',
                        content: (
                            <div>
                                <p>请使用微信扫描以下二维码完成支付：</p>
                                <img src={response.data.data.code_url} alt="支付二维码" style={{ width: '200px' }} />
                            </div>
                        ),
                        width: 400,
                    });
                }
            } else {
                showError(response.data.message);
            }
        } catch (error) {
            showError('创建支付订单失败');
        }
    };

    const columns = [
        {
            title: '订单号',
            dataIndex: 'order_id',
            key: 'order_id',
            width: 200,
        },
        {
            title: '充值数量',
            dataIndex: 'amount',
            key: 'amount',
            render: (amount) => `${amount} 额度`,
        },
        {
            title: '支付方式',
            dataIndex: 'payment_method',
            key: 'payment_method',
            render: (method) => {
                const methodMap = {
                    'alipay': { text: '支付宝', color: 'blue' },
                    'wechat': { text: '微信支付', color: 'green' },
                    'paypal': { text: 'PayPal', color: 'orange' },
                };
                const config = methodMap[method] || { text: method, color: 'default' };
                return <Tag color={config.color}>{config.text}</Tag>;
            },
        },
        {
            title: '状态',
            dataIndex: 'status',
            key: 'status',
            render: (status) => {
                const statusMap = {
                    'pending': { text: '待处理', color: 'orange' },
                    'processing': { text: '处理中', color: 'blue' },
                    'success': { text: '支付成功', color: 'green' },
                    'failed': { text: '支付失败', color: 'red' },
                };
                const config = statusMap[status] || { text: status, color: 'default' };
                return <Tag color={config.color}>{config.text}</Tag>;
            },
        },
        {
            title: '创建时间',
            dataIndex: 'created_at',
            key: 'created_at',
            render: (time) => new Date(time).toLocaleString(),
        },
        {
            title: '操作',
            key: 'action',
            render: (_, record) => (
                <Space>
                    {(record.status === 'pending' || record.status === 'processing') && (
                        <Button 
                            type="primary" 
                            size="small"
                            onClick={() => {
                                if (record.payment_method === 'alipay' && record.status === 'pending') {
                                    // 重新获取支付链接
                                    API.post('/api/user/pay', {
                                        amount: record.amount,
                                        payment_method: record.payment_method,
                                        top_up_code: record.top_up_code,
                                    }).then(response => {
                                        if (response.data.success) {
                                            window.open(response.data.url, '_blank');
                                        }
                                    });
                                }
                            }}
                            disabled={record.status === 'processing'}
                        >
                            {record.status === 'processing' ? '处理中' : '继续支付'}
                        </Button>
                    )}
                    <Button 
                        size="small"
                        onClick={() => {
                            Modal.info({
                                title: '订单详情',
                                content: (
                                    <div>
                                        <p><strong>订单号：</strong>{record.order_id}</p>
                                        <p><strong>充值数量：</strong>{record.amount} 额度</p>
                                        <p><strong>支付方式：</strong>{record.payment_method}</p>
                                        <p><strong>状态：</strong>{record.status}</p>
                                        <p><strong>创建时间：</strong>{new Date(record.created_at).toLocaleString()}</p>
                                        {record.top_up_code && (
                                            <p><strong>充值码：</strong>{record.top_up_code}</p>
                                        )}
                                    </div>
                                ),
                            });
                        }}
                    >
                        详情
                    </Button>
                </Space>
            ),
        },
    ];

    return (
        <div style={{ padding: '24px' }}>
            <Card>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
                    <Title level={4}>支付记录</Title>
                    <Button 
                        type="primary" 
                        onClick={() => setIsModalVisible(true)}
                    >
                        创建支付订单
                    </Button>
                </div>

                <Table
                    columns={columns}
                    dataSource={payments}
                    rowKey="id"
                    loading={loading}
                    pagination={{
                        current: currentPage,
                        pageSize: pageSize,
                        total: total,
                        showSizeChanger: true,
                        showQuickJumper: true,
                        showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
                        onChange: handlePageChange,
                        onShowSizeChange: handlePageChange,
                    }}
                />
            </Card>

            <Modal
                title="创建支付订单"
                visible={isModalVisible}
                onCancel={() => setIsModalVisible(false)}
                footer={null}
            >
                <Form
                    form={form}
                    layout="vertical"
                    onFinish={handleCreatePayment}
                >
                    <Form.Item
                        name="amount"
                        label="充值数量"
                        rules={[{ required: true, message: '请输入充值数量' }]}
                    >
                        <Input type="number" min={1} placeholder="请输入充值数量" />
                    </Form.Item>

                    <Form.Item
                        name="payment_method"
                        label="支付方式"
                        rules={[{ required: true, message: '请选择支付方式' }]}
                    >
                        <Select placeholder="请选择支付方式">
                            <Option value="alipay">支付宝</Option>
                            <Option value="wechat">微信支付</Option>
                            <Option value="paypal">PayPal</Option>
                        </Select>
                    </Form.Item>

                    <Form.Item
                        name="top_up_code"
                        label="充值码（可选）"
                    >
                        <Input placeholder="请输入充值码" />
                    </Form.Item>

                    <Form.Item>
                        <Space>
                            <Button type="primary" htmlType="submit">
                                创建订单
                            </Button>
                            <Button onClick={() => setIsModalVisible(false)}>
                                取消
                            </Button>
                        </Space>
                    </Form.Item>
                </Form>
            </Modal>
        </div>
    );
};

export default Payment; 