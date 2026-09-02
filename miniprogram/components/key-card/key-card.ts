Component({
  properties: {
    keyItem: {
      type: Object,
      value: {}
    }
  },
  methods: {
    onTap() {
      if (this.data.keyItem && this.data.keyItem.id) {
        this.triggerEvent('tapcard', { keyId: this.data.keyItem.id });
      }
    }
  }
});
