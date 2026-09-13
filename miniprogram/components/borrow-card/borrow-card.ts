Component({
  options: {
    addGlobalClass: true,
  },
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
      if (this.data.showReturnBtn && b?.canReturn && b.id) {
        this.triggerEvent('return', {
          id: b.id,
          keyId: b.keyId
        });
      }
    }
  }
});
