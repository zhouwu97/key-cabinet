Component({
  properties: {
    title: {
      type: String,
      value: ''
    },
    subTitle: {
      type: String,
      value: ''
    },
    actionText: {
      type: String,
      value: ''
    }
  },
  methods: {
    onAction() {
      this.triggerEvent('action');
    }
  }
});
