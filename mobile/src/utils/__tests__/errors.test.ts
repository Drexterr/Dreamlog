import { errorCode, errorMessage, httpStatus } from '../errors';

describe('error helpers', () => {
  it('reads messages from Errors and plain objects, with a fallback', () => {
    expect(errorMessage(new Error('boom'), 'fallback')).toBe('boom');
    expect(errorMessage({ message: 'from object' }, 'fallback')).toBe('from object');
    expect(errorMessage(new Error(''), 'fallback')).toBe('fallback');
    expect(errorMessage('a string', 'fallback')).toBe('fallback');
    expect(errorMessage(null, 'fallback')).toBe('fallback');
  });
  it('reads string codes only', () => {
    expect(errorCode({ code: 'ERR_REQUEST_CANCELED' })).toBe('ERR_REQUEST_CANCELED');
    expect(errorCode({ code: 42 })).toBeUndefined();
    expect(errorCode(undefined)).toBeUndefined();
  });
  it('reads axios response status', () => {
    expect(httpStatus({ response: { status: 402 } })).toBe(402);
    expect(httpStatus({ response: {} })).toBeUndefined();
    expect(httpStatus(new Error('x'))).toBeUndefined();
  });
});
