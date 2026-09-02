Component({
  properties: {
    reservation: {
      type: Object,
      value: {}
    },
    showActions: {
      type: Boolean,
      value: true
    }
  },
  methods: {
    onPickup() {
      const res = this.data.reservation;
      if (res && res.id) {
        this.triggerEvent('pickup', {
          id: res.id,
          keyId: res.keyId
        });
      }
    },
    onCancel() {
      const res = this.data.reservation;
      if (res && res.id) {
        this.triggerEvent('cancel', {
          id: res.id,
          keyId: res.keyId
        });
      }
    }
  }
});
