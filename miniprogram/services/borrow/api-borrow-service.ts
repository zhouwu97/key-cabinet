import { httpClient } from '../../api/http-client'
import { toISOTime, toQueryString, toTimestamp } from '../../api/serializers'
import { BorrowRecord, BorrowRecordStatus } from '../../models/borrow-record'
import { BorrowService } from './borrow-service'

function normalizeBorrowRecord(data: BorrowRecord & Record<string, unknown>): BorrowRecord {
  return {
    ...data,
    borrowedAt: data.borrowedAt === undefined ? undefined : toTimestamp(data.borrowedAt),
    expectedReturnAt: toTimestamp(data.expectedReturnAt),
    overdueAt: data.overdueAt === undefined ? undefined : toTimestamp(data.overdueAt),
    returnedAt: data.returnedAt === undefined ? undefined : toTimestamp(data.returnedAt),
  }
}

/** 真实后端借还记录服务；不在客户端推导或持久化借还业务事实。 */
export class ApiBorrowService implements BorrowService {
  async getUserBorrowRecords(_userId: string): Promise<BorrowRecord[]> {
    const records = await httpClient.request<Array<BorrowRecord & Record<string, unknown>>>({
      url: '/api/v1/me/borrow-records',
    })
    return records.map(normalizeBorrowRecord)
  }

  async getCurrentBorrows(userId: string): Promise<BorrowRecord[]> {
    const records = await this.getUserBorrowRecords(userId)
    return records.filter(record =>
      [BorrowRecordStatus.BORROWING, BorrowRecordStatus.BORROWED, BorrowRecordStatus.RETURNING].includes(
        record.status,
      ),
    )
  }

  async getBorrowRecordById(id: string): Promise<BorrowRecord | null> {
    try {
      const record = await httpClient.request<BorrowRecord & Record<string, unknown>>({
        url: `/api/v1/borrow-records/${encodeURIComponent(id)}`,
      })
      return normalizeBorrowRecord(record)
    } catch (error) {
      console.error(`Failed to get borrow record ${id}:`, error)
      return null
    }
  }

  async getActiveBorrowByKey(keyId: string): Promise<BorrowRecord | null> {
    const records = await httpClient.request<Array<BorrowRecord & Record<string, unknown>>>({
      url: `/api/v1/me/borrow-records${toQueryString({ keyId })}`,
    })
    return (
      records
        .map(normalizeBorrowRecord)
        .find(record =>
          [BorrowRecordStatus.BORROWING, BorrowRecordStatus.BORROWED, BorrowRecordStatus.RETURNING].includes(
            record.status,
          ),
        ) || null
    )
  }

  async createBorrowRecord(
    _userId: string,
    keyId: string,
    deviceId: string,
    slotId?: string,
    reservationId?: string,
    purpose?: string,
    expectedReturnAt?: number,
  ): Promise<BorrowRecord> {
    const record = await httpClient.request<BorrowRecord & Record<string, unknown>>({
      url: '/api/v1/borrow-records',
      method: 'POST',
      data: {
        keyId,
        deviceId,
        slotId,
        reservationId,
        purpose,
        expectedReturnAt: toISOTime(expectedReturnAt),
      },
    })
    return normalizeBorrowRecord(record)
  }

  async updateBorrowStatus(id: string, status: BorrowRecordStatus, notes?: string): Promise<void> {
    await httpClient.request<unknown>({
      url: `/api/v1/borrow-records/${encodeURIComponent(id)}/status`,
      method: 'PATCH',
      data: { status, notes },
    })
  }

  async completeBorrowRecord(id: string): Promise<void> {
    await httpClient.request<unknown>({
      url: `/api/v1/borrow-records/${encodeURIComponent(id)}/complete`,
      method: 'POST',
    })
  }

  async checkOverdue(): Promise<void> {
    await httpClient.request<unknown>({
      url: '/api/v1/borrow-records/check-overdue',
      method: 'POST',
    })
  }
}
