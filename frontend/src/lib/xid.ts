const alphabet = "0123456789abcdefghijklmnopqrstuv";

const decode = (value: string): Uint8Array => {
  const normalized = value.toLowerCase();
  if (!/^[a-v0-9]{20}$/.test(normalized)) {
    throw new Error("invalid xid: expected 20 lowercase base32hex characters");
  }
  const bytes = new Uint8Array(12);
  let buffer = 0;
  let bits = 0;
  let offset = 0;
  for (const character of normalized) {
    buffer = (buffer << 5) | alphabet.indexOf(character);
    bits += 5;
    if (bits >= 8) {
      bits -= 8;
      bytes[offset++] = (buffer >> bits) & 0xff;
    }
  }
  return bytes;
};

export class XID {
  private readonly timestamp: Date;
  private readonly machineId: number;
  private readonly processId: number;
  private readonly counter: number;

  constructor(timestamp: Date, machineId: number, processId: number, counter: number) {
    this.timestamp = timestamp;
    this.machineId = machineId;
    this.processId = processId;
    this.counter = counter;
  }

  static nilId(): XID {
    return new XID(new Date(0), 0, 0, 0);
  }

  static parse(value: string): XID {
    const bytes = decode(value);
    const timestamp = ((bytes[0] << 24) | (bytes[1] << 16) | (bytes[2] << 8) | bytes[3]) >>> 0;
    return new XID(
      new Date(timestamp * 1000),
      (bytes[4] << 16) | (bytes[5] << 8) | bytes[6],
      (bytes[7] << 8) | bytes[8],
      (bytes[9] << 16) | (bytes[10] << 8) | bytes[11],
    );
  }

  timestampMs(): number {
    return this.timestamp.getTime();
  }
  counterValue(): number {
    return this.counter;
  }
  machineIdValue(): number {
    return this.machineId;
  }
  processIdValue(): number {
    return this.processId;
  }

  compare(other: XID): number {
    return (
      this.timestampMs() - other.timestampMs() ||
      this.machineId - other.machineId ||
      this.processId - other.processId ||
      this.counter - other.counter
    );
  }

  isNil(): boolean {
    return this.compare(XID.nilId()) === 0;
  }
}
