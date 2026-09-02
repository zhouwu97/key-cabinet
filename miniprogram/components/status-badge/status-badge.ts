Component({
  properties: {
    text: {
      type: String,
      value: ''
    },
    tone: {
      type: String,
      value: 'gray' // 'green' | 'blue' | 'orange' | 'red' | 'gray'
    },
    dot: {
      type: Boolean,
      value: true
    }
  }
});
