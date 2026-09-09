Component({
  options: {
    addGlobalClass: true,
  },
  properties: {
    icon: {
      type: String,
      value: '/assets/icons/icon-warning.svg',
    },
    title: {
      type: String,
      value: '遇到异常情况',
    },
    message: {
      type: String,
      value: '暂时无法获取数据，请稍后重试',
    },
    retryText: {
      type: String,
      value: '重新加载',
    },
  },
  methods: {
    onRetry() {
      this.triggerEvent('retry');
    },
  },
});
