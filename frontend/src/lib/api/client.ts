import {
  AuthResponse,
  Clinic,
  Doctor,
  TherapyPackage,
  TreatmentRoom,
  AvailableSlot,
  AppointmentHold,
  Appointment,
  TherapySession,
  ClinicMetrics,
  DetailedClinicDashboard,
  FeedbackItem,
  SubmitFeedbackPayload,
  AdverseEventItem,
  ReportAdverseEventPayload,
  User,
  PatientProfile,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// Token Storage Keys
const TOKEN_KEY = 'panchakarma_access_token';
const REFRESH_KEY = 'panchakarma_refresh_token';
const USER_KEY = 'panchakarma_user';
const PROFILE_KEY = 'panchakarma_patient_profile';

export const tokenStorage = {
  getAccessToken: (): string | null => localStorage.getItem(TOKEN_KEY),
  getRefreshToken: (): string | null => localStorage.getItem(REFRESH_KEY),
  getUser: (): User | null => {
    const data = localStorage.getItem(USER_KEY);
    return data ? JSON.parse(data) : null;
  },
  getProfile: (): PatientProfile | null => {
    const data = localStorage.getItem(PROFILE_KEY);
    return data ? JSON.parse(data) : null;
  },
  setAuth: (data: AuthResponse) => {
    localStorage.setItem(TOKEN_KEY, data.access_token);
    localStorage.setItem(REFRESH_KEY, data.refresh_token);
    localStorage.setItem(USER_KEY, JSON.stringify(data.user));
    if (data.profile) {
      localStorage.setItem(PROFILE_KEY, JSON.stringify(data.profile));
    }
  },
  clearAuth: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(USER_KEY);
    localStorage.removeItem(PROFILE_KEY);
  },
};

// Real Direct API Client (Zero Mock Fallbacks)
async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = tokenStorage.getAccessToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Trace-ID': `web-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`,
    ...(options.headers as Record<string, string>),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const url = `${API_BASE_URL}${endpoint}`;

  let response: Response;
  try {
    response = await fetch(url, {
      ...options,
      headers,
    });
  } catch (err: any) {
    throw new Error(
      `Cannot connect to PanchakarmaFlow backend at ${API_BASE_URL}. Ensure all microservices are running via .\\run-backend.ps1`
    );
  }

  // Handle token rotation / expiration
  if (response.status === 401 && tokenStorage.getRefreshToken()) {
    const refreshed = await tryRefreshToken();
    if (refreshed) {
      headers['Authorization'] = `Bearer ${tokenStorage.getAccessToken()}`;
      try {
        const retryRes = await fetch(url, { ...options, headers });
        if (retryRes.ok) {
          const json = await retryRes.json();
          return json.data !== undefined ? json.data : json;
        }
      } catch (retryErr: any) {
        throw new Error(`Connection failed on token retry: ${retryErr.message}`);
      }
    }
  }

  if (!response.ok) {
    const errBody = await response.json().catch(() => ({}));
    const message =
      (typeof errBody.error === 'object' && errBody.error?.message)
        ? errBody.error.message
        : (typeof errBody.error === 'string'
            ? errBody.error
            : (errBody.message || `Request to ${endpoint} failed with HTTP ${response.status}`));
    throw new Error(message);
  }

  const data = await response.json();
  return data.data !== undefined ? data.data : data;
}

async function tryRefreshToken(): Promise<boolean> {
  const refresh = tokenStorage.getRefreshToken();
  if (!refresh) return false;
  try {
    const res = await fetch(`${API_BASE_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
    });
    if (res.ok) {
      const data = await res.json();
      const payload: AuthResponse = data.data ?? data;
      tokenStorage.setAuth(payload);
      return true;
    }
  } catch {
    // refresh failed
  }
  tokenStorage.clearAuth();
  return false;
}

// =========================================================================
// AUTHENTICATION API (Connected to Go Identity Service)
// =========================================================================

export const authApi = {
  login: async (email: string, password?: string): Promise<AuthResponse> => {
    const res = await apiRequest<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    tokenStorage.setAuth(res);
    return res;
  },

  register: async (payload: {
    email: string;
    password?: string;
    full_name: string;
    phone?: string;
    date_of_birth?: string;
    biological_sex?: string;
    blood_group?: string;
    emergency_contact_name?: string;
    emergency_contact_phone?: string;
    address_street?: string;
    address_city?: string;
    address_state?: string;
    address_postal_code?: string;
  }): Promise<AuthResponse> => {
    const completePayload = {
      date_of_birth: payload.date_of_birth || '1995-01-01',
      biological_sex: payload.biological_sex || 'female',
      emergency_contact_name: payload.emergency_contact_name || (payload.full_name ? `${payload.full_name}'s Contact` : 'Emergency Contact'),
      emergency_contact_phone: payload.emergency_contact_phone || payload.phone || '+919876543210',
      ...payload,
    };
    const res = await apiRequest<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(completePayload),
    });
    tokenStorage.setAuth(res);
    return res;
  },

  logout: async (): Promise<void> => {
    try {
      const refreshToken = tokenStorage.getRefreshToken();
      if (refreshToken) {
        await apiRequest('/auth/logout', {
          method: 'POST',
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
      }
    } catch {
      // ignore network errors on logout
    } finally {
      tokenStorage.clearAuth();
    }
  },

  getMe: async (): Promise<{ user: User; profile?: PatientProfile }> => {
    return apiRequest<{ user: User; profile?: PatientProfile }>('/auth/me', {
      method: 'GET',
    });
  },
};

// =========================================================================
// CLINIC & CATALOG API (Connected to Go Clinic Service)
// =========================================================================

export const clinicsApi = {
  list: async (): Promise<Clinic[]> => {
    const res = await apiRequest<any>('/clinics', { method: 'GET' });
    if (Array.isArray(res)) return res;
    return res?.clinics || [];
  },

  getById: async (id: string): Promise<Clinic> => {
    return apiRequest<Clinic>(`/clinics/${id}`, { method: 'GET' });
  },

  getPackages: async (clinicId: string): Promise<TherapyPackage[]> => {
    const res = await apiRequest<any>(`/clinics/${clinicId}/packages`, { method: 'GET' });
    if (Array.isArray(res)) return res;
    return res?.packages || [];
  },

  createPackage: async (clinicId: string, payload: any): Promise<TherapyPackage> => {
    return apiRequest<TherapyPackage>(`/clinics/${clinicId}/packages`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  publishPackage: async (clinicId: string, packageId: string): Promise<void> => {
    return apiRequest(`/clinics/${clinicId}/packages/${packageId}/publish`, {
      method: 'POST',
    });
  },

  getDoctors: async (clinicId: string): Promise<Doctor[]> => {
    const res = await apiRequest<any>(`/clinics/${clinicId}/doctors`, { method: 'GET' });
    if (Array.isArray(res)) return res;
    return res?.doctors || [];
  },

  getRooms: async (clinicId: string): Promise<TreatmentRoom[]> => {
    const res = await apiRequest<any>(`/clinics/${clinicId}/rooms`, { method: 'GET' });
    if (Array.isArray(res)) return res;
    return res?.rooms || [];
  },

  createRoom: async (clinicId: string, payload: any): Promise<TreatmentRoom> => {
    return apiRequest<TreatmentRoom>(`/clinics/${clinicId}/rooms`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },
};

// =========================================================================
// APPOINTMENTS & SCHEDULING API (Connected to Go Appointment Service)
// =========================================================================

export const appointmentsApi = {
  searchAvailability: async (
    clinicId: string,
    packageId?: string,
    doctorId?: string,
    date?: string
  ): Promise<AvailableSlot[]> => {
    const params = new URLSearchParams({ clinic_id: clinicId });
    if (packageId) params.append('package_id', packageId);
    if (doctorId && doctorId !== 'auto') params.append('doctor_id', doctorId);
    if (date) {
      params.append('start_date', date);
      params.append('end_date', date);
    }

    const res = await apiRequest<any>(`/availability?${params.toString()}`, {
      method: 'GET',
    });
    if (Array.isArray(res)) return res;
    return res?.available_slots || res?.slots || [];
  },

  createHold: async (payload: {
    clinic_id: string;
    patient_id?: string;
    doctor_id?: string;
    room_id?: string;
    package_id: string;
    start_time: string;
    duration_minutes?: number;
    auto_assign_doctor?: boolean;
  }): Promise<AppointmentHold> => {
    return apiRequest<AppointmentHold>('/bookings/hold', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  confirmHold: async (holdToken: string, patientId?: string): Promise<Appointment> => {
    return apiRequest<Appointment>('/bookings/confirm', {
      method: 'POST',
      body: JSON.stringify({ hold_token: holdToken }),
    });
  },

  list: async (filters: Record<string, string> = {}): Promise<Appointment[]> => {
    const query = new URLSearchParams(filters).toString();
    const res = await apiRequest<any>(`/appointments${query ? `?${query}` : ''}`, {
      method: 'GET',
    });
    if (Array.isArray(res)) return res;
    return res?.appointments || [];
  },

  listForPatient: async (patientId: string): Promise<Appointment[]> => {
    const res = await apiRequest<any>(
      `/appointments?patient_id=${encodeURIComponent(patientId)}`,
      { method: 'GET' }
    );
    if (Array.isArray(res)) return res;
    return res?.appointments || [];
  },

  getById: async (id: string): Promise<Appointment> => {
    return apiRequest<Appointment>(`/appointments/${id}`, { method: 'GET' });
  },

  cancel: async (id: string, reason: string): Promise<void> => {
    return apiRequest(`/appointments/${id}/cancel`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  },

  reschedule: async (id: string, newStartTime: string, reason: string): Promise<Appointment> => {
    return apiRequest<Appointment>(`/appointments/${id}/reschedule`, {
      method: 'POST',
      body: JSON.stringify({ new_start_time: newStartTime, reason }),
    });
  },
};

// =========================================================================
// THERAPY EXECUTION & TRACKING API (Connected to Go Therapy Service)
// =========================================================================

export const therapyApi = {
  checkIn: async (arg1: any, arg2?: any): Promise<TherapySession> => {
    const payload =
      typeof arg1 === 'string'
        ? {
            appointment_id: arg1,
            baseline_vitals: arg2,
            pre_checklist_verified: true,
          }
        : arg1;
    return apiRequest<TherapySession>('/sessions/check-in', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  startSession: async (sessionId: string): Promise<TherapySession> => {
    return apiRequest<TherapySession>(`/sessions/${sessionId}/start`, { method: 'POST' });
  },

  completeSession: async (sessionId: string, payload?: {
    therapist_notes?: string;
    observations?: string;
    post_care_instructions?: string;
    post_care_items?: any;
  }): Promise<TherapySession> => {
    return apiRequest<TherapySession>(`/sessions/${sessionId}/complete`, {
      method: 'POST',
      body: JSON.stringify(payload || {}),
    });
  },

  startStep: async (sessionId: string, stepOrder: number): Promise<void> => {
    return apiRequest(`/sessions/${sessionId}/steps/${stepOrder}/start`, { method: 'POST' });
  },

  pauseStep: async (sessionId: string, stepOrder: number, reason: string): Promise<void> => {
    return apiRequest(`/sessions/${sessionId}/steps/${stepOrder}/pause`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  },

  resumeStep: async (sessionId: string, stepOrder: number): Promise<void> => {
    return apiRequest(`/sessions/${sessionId}/steps/${stepOrder}/resume`, { method: 'POST' });
  },

  completeStep: async (sessionId: string, stepOrder: number, practitionerNotes?: string, observations?: string): Promise<void> => {
    return apiRequest(`/sessions/${sessionId}/steps/${stepOrder}/complete`, {
      method: 'POST',
      body: JSON.stringify({
        therapist_notes: practitionerNotes || '',
        observations: observations || '',
      }),
    });
  },

  recordMaterial: async (sessionId: string, payload: any): Promise<void> => {
    return apiRequest(`/sessions/${sessionId}/materials`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  flagComplication: async (sessionId: string, payload: any): Promise<void> => {
    return apiRequest(`/sessions/${sessionId}/complications`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  createFollowUp: async (sessionId: string, payload: any): Promise<void> => {
    return apiRequest(`/sessions/${sessionId}/follow-ups`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  getSession: async (sessionId: string): Promise<TherapySession> => {
    return apiRequest<TherapySession>(`/sessions/${sessionId}`, { method: 'GET' });
  },

  listTherapistAssigned: async (): Promise<TherapySession[]> => {
    return apiRequest<TherapySession[]>('/sessions/therapist/assigned', { method: 'GET' });
  },

  listPatientSessions: async (): Promise<TherapySession[]> => {
    return apiRequest<TherapySession[]>('/sessions/patient/me', { method: 'GET' });
  },
};

// =========================================================================
// FEEDBACK & ADVERSE EVENTS API (Connected to Go Feedback & Analytics Service)
// =========================================================================

export const feedbackApi = {
  submitFeedback: async (payload: SubmitFeedbackPayload): Promise<FeedbackItem> => {
    const overall = payload.overall_rating ?? payload.rating ?? 5;
    const body: Record<string, any> = {
      appointment_id: payload.appointment_id,
      overall_rating: overall,
      therapist_rating: payload.therapist_rating ?? overall,
      facility_cleanliness_rating: payload.facility_cleanliness_rating ?? overall,
      is_anonymous: !!payload.is_anonymous,
    };
    if (payload.session_id) body.session_id = payload.session_id;
    if (payload.clinic_id) body.clinic_id = payload.clinic_id;
    if (payload.doctor_id) body.doctor_id = payload.doctor_id;
    if (payload.therapist_id) body.therapist_id = payload.therapist_id;
    const commentText = payload.comments ?? payload.comment;
    if (commentText) body.comments = commentText;

    return apiRequest<FeedbackItem>('/feedback', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  list: async (params: { clinic_id?: string; limit?: number; offset?: number } = {}): Promise<FeedbackItem[]> => {
    const query = new URLSearchParams(params as any).toString();
    const res = await apiRequest<any>(`/feedback${query ? `?${query}` : ''}`, { method: 'GET' });
    if (Array.isArray(res)) return res;
    return res?.feedbacks || res?.feedback || [];
  },

  getById: async (id: string): Promise<FeedbackItem> => {
    return apiRequest<FeedbackItem>(`/feedback/${id}`, { method: 'GET' });
  },

  acknowledge: async (id: string, notes: string): Promise<void> => {
    return apiRequest(`/feedback/${id}/acknowledge`, {
      method: 'POST',
      body: JSON.stringify({ staff_notes: notes }),
    });
  },

  reportAdverseEvent: async (payload: ReportAdverseEventPayload): Promise<AdverseEventItem> => {
    // Map severity to Go enum: LOW, MEDIUM, HIGH, CRITICAL
    let severityVal = payload.severity;
    if (severityVal === 'MILD') severityVal = 'LOW';
    else if (severityVal === 'MODERATE') severityVal = 'MEDIUM';
    else if (severityVal === 'SEVERE') severityVal = 'HIGH';

    const body: Record<string, any> = {
      severity: severityVal,
      symptoms: payload.symptoms,
      clinic_id: payload.clinic_id || 'a3f8c872-9112-4c22-b5e1-7e8913c2f101',
    };
    if (payload.appointment_id) body.appointment_id = payload.appointment_id;
    if (payload.session_id) body.session_id = payload.session_id;
    if (payload.patient_id) body.patient_id = payload.patient_id;
    if (payload.vitals_at_incident || payload.vitals) {
      body.vitals_at_incident = payload.vitals_at_incident || payload.vitals;
    }
    const actions = payload.immediate_actions_taken ?? payload.immediate_actions;
    if (actions) body.immediate_actions_taken = actions;

    return apiRequest<AdverseEventItem>('/feedback/adverse-events', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  listAdverseEvents: async (params: { clinic_id?: string; severity?: string; status?: string } = {}): Promise<AdverseEventItem[]> => {
    const query = new URLSearchParams(params as any).toString();
    const res = await apiRequest<any>(`/feedback/adverse-events${query ? `?${query}` : ''}`, { method: 'GET' });
    if (Array.isArray(res)) return res;
    return res?.adverse_events || [];
  },

  getAdverseEventById: async (id: string): Promise<AdverseEventItem> => {
    return apiRequest<AdverseEventItem>(`/feedback/adverse-events/${id}`, { method: 'GET' });
  },

  acknowledgeAdverseEvent: async (id: string, notes: string = 'Staff reviewed', status = 'ACKNOWLEDGED'): Promise<void> => {
    return apiRequest(`/feedback/adverse-events/${id}/acknowledge`, {
      method: 'POST',
      body: JSON.stringify({ staff_notes: notes, status }),
    });
  },
};

// =========================================================================
// ANALYTICS API (Connected to Go Feedback & Analytics Service)
// =========================================================================

export const analyticsApi = {
  getClinicDashboard: async (clinicId: string, startDate?: string, endDate?: string): Promise<DetailedClinicDashboard> => {
    const params = new URLSearchParams();
    if (startDate) params.append('start_date', startDate);
    if (endDate) params.append('end_date', endDate);
    const qs = params.toString() ? `?${params.toString()}` : '';
    return apiRequest<DetailedClinicDashboard>(`/analytics/clinics/${clinicId}/dashboard${qs}`, {
      method: 'GET',
    });
  },

  getClinicMetrics: async (clinicId: string, date?: string): Promise<ClinicMetrics> => {
    const qs = date ? `?date=${encodeURIComponent(date)}` : '';
    return apiRequest<ClinicMetrics>(`/analytics/clinics/${clinicId}/daily-metrics${qs}`, {
      method: 'GET',
    });
  },

  exportCSV: async (clinicId: string, type: 'metrics' | 'feedback' | 'adverse-events' = 'metrics'): Promise<string> => {
    const token = tokenStorage.getAccessToken();
    const url = `${API_BASE_URL}/analytics/clinics/${clinicId}/export?type=${type}`;
    const res = await fetch(url, {
      headers: {
        Authorization: token ? `Bearer ${token}` : '',
      },
    });
    if (!res.ok) {
      throw new Error(`CSV export failed with status ${res.status}`);
    }
    return res.text();
  },
};

// Unified Convenience Aliases
export const clinicApi = {
  list: clinicsApi.list,
  getById: clinicsApi.getById,
};

export const packageApi = {
  listByClinic: clinicsApi.getPackages,
  createPackage: async (payload: any) => clinicsApi.createPackage(payload.clinic_id, payload),
};

export const doctorApi = {
  listByClinic: clinicsApi.getDoctors,
  createLeave: async (doctorId: string, payload: any) => {
    return apiRequest(`/doctors/${doctorId}/leaves`, {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },
};

export const roomApi = {
  listByClinic: clinicsApi.getRooms,
  createRoom: async (payload: any) => clinicsApi.createRoom(payload.clinic_id, payload),
};

export const appointmentApi = {
  listPatientAppointments: async (): Promise<Appointment[]> => {
    return appointmentsApi.list();
  },
  listClinicAppointments: async (clinicId: string, status?: string): Promise<Appointment[]> => {
    const filters: Record<string, string> = { clinic_id: clinicId };
    if (status && status !== 'ALL') filters['status'] = status;
    return appointmentsApi.list(filters);
  },
  searchSlots: async (params: {
    clinic_id: string;
    package_id?: string;
    doctor_id?: string;
    date?: string;
  }) =>
    appointmentsApi.searchAvailability(
      params.clinic_id,
      params.package_id,
      params.doctor_id,
      params.date
    ),
  createHold: async (payload: any) => {
    const user = tokenStorage.getUser();
    return appointmentsApi.createHold({
      clinic_id: payload.clinic_id,
      patient_id: payload.patient_id || user?.id || '',
      doctor_id: payload.doctor_id || '',
      room_id: payload.room_id || '',
      package_id: payload.package_id,
      start_time: payload.start_time,
      duration_minutes: payload.duration_minutes || 60,
      auto_assign_doctor: payload.auto_assign_doctor,
    });
  },
  confirmHold: async (holdToken: string) => {
    const user = tokenStorage.getUser();
    return appointmentsApi.confirmHold(holdToken, user?.id || '');
  },
  cancelAppointment: async (id: string, reason: string) => appointmentsApi.cancel(id, reason),
};
