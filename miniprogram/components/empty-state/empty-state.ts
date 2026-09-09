Component({
  options: {
    addGlobalClass: true,
  },
  properties: {
    icon: {
      type: String,
      value: '/assets/icons/icon-empty.svg',
      observer(val: string) {
        this.setData({
          isImageIcon: Boolean(val && (val.includes('/') || val.includes('.svg') || val.includes('.png'))),
        });
      },
    },
    title: {
      type: String,
      value: '暂无数据',
    },
    description: {
      type: String,
      value: '',
    },
    actionText: {
      type: String,
      value: '',
    },
  },
  data: {
    isImageIcon: true,
  },
  attached() {
    const icon = this.data.icon || '';
    this.setData({
      isImageIcon: Boolean(icon && (icon.includes('/') || icon.includes('.svg') || icon.includes('.png'))),
    });
  },
  methods: {
    onAction() {
      this.triggerEvent('action');
    },
  },
});
