Component({
  properties: {
    name: {
      type: String,
      value: '智能钥匙柜'
    },
    statusLabel: {
      type: String,
      value: '在线正常'
    },
    tone: {
      type: String,
      value: 'green'
    },
    location: {
      type: String,
      value: ''
    },
    availableCount: {
      type: Number,
      value: 0
    },
    totalCount: {
      type: Number,
      value: 0
    }
  }
});
