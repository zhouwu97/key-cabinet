Component({
  properties: {
    steps: {
      type: Array,
      value: []
    },
    currentStep: {
      type: Number,
      value: 1
    },
    totalSteps: {
      type: Number,
      value: 6
    },
    actionTitle: {
      type: String,
      value: '执行进度'
    }
  }
});
