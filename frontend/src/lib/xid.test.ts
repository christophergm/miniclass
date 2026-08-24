import { describe, expect, it } from 'vitest'
import { XID } from './xid'

describe('XID', () => {
  it('parses the nil identifier', () => {
    const xid = XID.parse('00000000000000000000')
    expect(xid.isNil()).toBe(true)
    expect(xid.timestampMs()).toBe(0)
    expect(xid.machineIdValue()).toBe(0)
    expect(xid.processIdValue()).toBe(0)
    expect(xid.counterValue()).toBe(0)
  })

  it('compares identifier components in order', () => {
    const earlier = XID.parse('01k2m1h0000000000000')
    const later = XID.parse('01k2m1h000000000000g')
    expect(earlier.isNil()).toBe(false)
    expect(earlier.compare(later)).toBeLessThan(0)
    expect(later.compare(earlier)).toBeGreaterThan(0)
    expect(earlier.compare(earlier)).toBe(0)
  })

  it('rejects malformed identifiers', () => {
    expect(() => XID.parse('not-an-xid')).toThrow('invalid xid')
    expect(() => XID.parse('0000000000000000000Z')).toThrow('invalid xid')
  })
})
