import { httpClient } from '../../api/http-client'
import { toISOTime, toQueryString, toTimestamp } from '../../api/serializers'
import { Reservation, ReservationStatus } from '../../models/reservation'
import {
  CreateReservationParams,
  ReservationService,
} from './reservation-service'

function normalizeReservation(data: Reservation & Record<string, unknown>): Reservation {
  return {
    ...data,
    createdAt: toTimestamp(data.createdAt),
    pickupWindowStart: toTimestamp(data.pickupWindowStart),
    pickupWindowEnd: toTimestamp(data.pickupWindowEnd),
    expectedReturnAt: toTimestamp(data.expectedReturnAt),
    approvedAt: data.approvedAt === undefined ? undefined : toTimestamp(data.approvedAt),
    usedAt: data.usedAt === undefined ? undefined : toTimestamp(data.usedAt),
    cancelledAt: data.cancelledAt === undefined ? undefined : toTimestamp(data.cancelledAt),
  }
}

/** 真实后端预约服务，用户身份始终由 JWT 决定，不把 userId 发给服务器。 */
export class ApiReservationService implements ReservationService {
  async createReservation(params: CreateReservationParams): Promise<Reservation> {
    const reservation = await httpClient.request<Reservation & Record<string, unknown>>({
      url: '/api/v1/reservations',
      method: 'POST',
      data: {
        keyId: params.keyId,
        pickupWindowStart: toISOTime(params.pickupWindowStart),
        pickupWindowEnd: toISOTime(params.pickupWindowEnd),
        expectedReturnAt: toISOTime(params.expectedReturnAt),
        expectedDuration: params.expectedDuration,
        purpose: params.purpose,
      },
    })
    return normalizeReservation(reservation)
  }

  async getUserReservations(_userId: string): Promise<Reservation[]> {
    const reservations = await httpClient.request<Array<Reservation & Record<string, unknown>>>({
      url: '/api/v1/me/reservations',
    })
    return reservations.map(normalizeReservation)
  }

  async getReservationById(id: string): Promise<Reservation | null> {
    try {
      const reservation = await httpClient.request<Reservation & Record<string, unknown>>({
        url: `/api/v1/reservations/${encodeURIComponent(id)}`,
      })
      return normalizeReservation(reservation)
    } catch (error) {
      console.error(`Failed to get reservation ${id}:`, error)
      return null
    }
  }

  async cancelReservation(id: string): Promise<void> {
    await httpClient.request<unknown>({
      url: `/api/v1/reservations/${encodeURIComponent(id)}/cancel`,
      method: 'POST',
    })
  }

  async canReserveKey(keyId: string, windowStart?: number, windowEnd?: number): Promise<boolean> {
    const result = await httpClient.request<boolean | { available: boolean }>({
      url: `/api/v1/keys/${encodeURIComponent(keyId)}/availability${toQueryString({
        pickupWindowStart: toISOTime(windowStart),
        pickupWindowEnd: toISOTime(windowEnd),
      })}`,
    })
    return typeof result === 'boolean' ? result : result.available
  }

  async getActiveReservation(userId: string, keyId?: string): Promise<Reservation | null> {
    const reservations = await this.getUserReservations(userId)
    const now = Date.now()
    return (
      reservations.find(
        reservation =>
          (!keyId || reservation.keyId === keyId) &&
          (reservation.status === ReservationStatus.ACTIVE ||
            (reservation.status === ReservationStatus.APPROVED &&
              now >= reservation.pickupWindowStart &&
              now <= reservation.pickupWindowEnd)),
      ) || null
    )
  }

  async markReservationUsed(id: string): Promise<void> {
    await httpClient.request<unknown>({
      url: `/api/v1/reservations/${encodeURIComponent(id)}/use`,
      method: 'POST',
    })
  }
}
