const assert = require('assert');
const { freeShippingState, formatMoney } = require('./cart-progress');

assert.deepStrictEqual(freeShippingState(50000, 100000), {
  unlocked: false, percent: 50, remainingCents: 50000, disabled: false,
});
assert.strictEqual(formatMoney(123450, '₹'), '₹1,234.50');

console.log('basic tests passed');
