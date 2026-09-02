Component({
  properties: {
    borrow: {
      type: Object,
      value: {}
    },
    showReturnBtn: {
      type: Boolean,
      value: true
    }
  },
  methods: {
    onReturn() {
      const b = this.data.borrow;
      if (b && b.id) {
        this.triggerEvent('return', {
          id: b.id,
          keyId: b.keyId
        });
      }
    }
  }
});
