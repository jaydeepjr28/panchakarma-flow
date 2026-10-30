// Standardized API Types aligned with Go Microservices Contracts

export type UserRole = 'patient' | 'clinic_admin' | 'therapist' | 'doctor' | 'super_admin';

export interface User {
  id: string;
  email: string;
  phone?: string;
  role: UserRole;
  clinic_id?: string;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface PatientProfile {
  id: string;
  user_id: string;
  full_name: string;
  date_of_birth?: string;
  biological_sex?: string;
  blood_group?: string;
  emergency_contact_name?: string;
  emergency_contact_phone?: string;
  address_street?: string;
  address_city?: string;
  address_state?: string;
  address_postal_code?: string;
  allergies?: string[];
  medical_history?: string;
  prakriti_type?: string;
  created_at?: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user: User;
  profile?: PatientProfile;
}

export interface Clinic {
  id: string;
  name: string;
  code: string;
  address: string;
  phone: string;
  email: string;
  hours?: string;
  time_zone?: string;
  languages?: string[];
  description?: string;
  is_active: boolean;
  rating?: number;
  review_count?: number;
  image_url?: string;
}

export interface Doctor {
  id: string;
  user_id: string;
  clinic_id: string;
  full_name: string;
  specialties: string[];
  qualifications: string;
  bio?: string;
  max_daily_capacity?: number;
  is_active: boolean;
}

export interface DoctorSchedule {
  id: string;
  doctor_id: string;
  clinic_id: string;
  day_of_week: number; // 0=Sunday, 1=Monday...
  start_time: string;  // "09:00"
  end_time: string;    // "17:00"
  slot_duration_minutes: number;
}

export interface DoctorLeave {
  id: string;
  doctor_id: string;
  start_date: string;
  end_date: string;
  reason?: string;
}

export interface TreatmentRoom {
  id: string;
  clinic_id: string;
  room_number: string;
  name: string;
  room_type: string;
  capacity: number;
  is_active: boolean;
}

export interface PackageStep {
  step_order: number;
  name: string;
  description?: string;
  duration_minutes: number;
  required_room_type?: string;
}

export interface TherapyPackage {
  id: string;
  clinic_id: string;
  name: string;
  code: string;
  description: string;
  total_sessions: number;
  recommended_interval_days?: number;
  duration_minutes: number;
  price?: number;
  is_published: boolean;
  version?: number;
  steps?: PackageStep[];
  pre_care_checklist?: string[];
  post_care_instructions?: string;
}

export interface AvailableSlot {
  slot_id: string;
  clinic_id: string;
  package_id?: string;
  doctor_id?: string;
  doctor_name?: string;
  room_id?: string;
  room_name?: string;
  start_time: string;
  end_time: string;
  duration_minutes: number;
  is_available: boolean;
}

export interface AppointmentHold {
  hold_token: string;
  hold_id: string;
  clinic_id: string;
  patient_id: string;
  doctor_id: string;
  room_id: string;
  package_id: string;
  start_time: string;
  end_time: string;
  duration_minutes: number;
  expires_at: string;
}

export type AppointmentStatus =
  | 'PENDING'
  | 'CONFIRMED'
  | 'CHECKED_IN'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'CANCELLED'
  | 'RESCHEDULED'
  | 'NO_SHOW';

export interface Appointment {
  id: string;
  clinic_id: string;
  patient_id: string;
  patient_name?: string;
  doctor_id: string;
  doctor_name?: string;
  room_id: string;
  room_name?: string;
  package_id: string;
  package_name?: string;
  start_time: string;
  end_time: string;
  duration_minutes: number;
  status: AppointmentStatus;
  notes?: string;
  created_at: string;
}

export interface BaselineVitals {
  bp_systolic: number;
  bp_diastolic: number;
  pulse: number;
  temperature_c: number;
}

export type SessionStatus =
  | 'CHECKED_IN'
  | 'IN_PROGRESS'
  | 'COMPLICATION_ALERT'
  | 'COMPLETED'
  | 'ABORTED_CLINICAL';

export type StepStatus =
  | 'NOT_STARTED'
  | 'IN_PROGRESS'
  | 'PAUSED'
  | 'COMPLETED'
  | 'SKIPPED';

export interface StepExecution {
  id: string;
  session_id: string;
  step_id?: string;
  step_order: number;
  name: string;
  planned_duration_minutes: number;
  actual_duration_seconds: number;
  status: StepStatus;
  started_at?: string;
  paused_at?: string;
  completed_at?: string;
  therapist_notes?: string;
  observations?: string;
}

export interface SessionMaterial {
  id: string;
  session_id: string;
  material_name: string;
  quantity: number;
  unit: string;
  batch_number?: string;
  notes?: string;
  recorded_by?: string;
  created_at?: string;
}

export interface ComplicationEvent {
  id: string;
  session_id: string;
  reporter_id: string;
  reporter_role: string;
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
  symptoms: string;
  vitals_at_incident?: BaselineVitals;
  immediate_actions_taken?: string;
  status: 'UNRESOLVED' | 'DOCTOR_REVIEWED' | 'RESOLVED';
  attending_doctor_id?: string;
  doctor_notes?: string;
  doctor_cleared_at?: string;
  created_at: string;
}

export interface SessionFollowUp {
  id: string;
  session_id: string;
  recommended_by: string;
  recommended_package_id?: string;
  recommended_interval_days?: number;
  recommended_date?: string;
  instructions: string;
  linked_appointment_id?: string;
  status: 'PENDING' | 'SCHEDULED' | 'DISMISSED';
  created_at: string;
}

export interface SessionAuditLog {
  id: string;
  session_id: string;
  field_name: string;
  old_value?: string;
  new_value: string;
  reason: string;
  corrected_by: string;
  created_at: string;
}

export interface TherapySession {
  id: string;
  appointment_id: string;
  patient_id: string;
  patient_name?: string;
  clinic_id: string;
  doctor_id: string;
  therapist_id: string;
  room_id: string;
  package_id: string;
  package_name?: string;
  package_version: number;
  status: SessionStatus;
  baseline_vitals?: BaselineVitals;
  pre_checklist_verified: boolean;
  pre_checklist_data?: Record<string, any>;
  therapist_notes?: string;
  observations?: string;
  post_care_instructions?: string;
  post_care_dispatched: boolean;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
  steps?: StepExecution[];
  materials?: SessionMaterial[];
  follow_ups?: SessionFollowUp[];
}

export interface NotificationItem {
  id: string;
  user_id: string;
  title: string;
  message: string;
  type: 'BOOKING' | 'REMINDER' | 'CHECKLIST' | 'EMERGENCY' | 'FOLLOW_UP';
  is_read: boolean;
  created_at: string;
}

export interface FeedbackItem {
  id: string;
  appointment_id: string;
  patient_id: string;
  patient_name?: string;
  clinic_id: string;
  session_id?: string;
  rating: number;
  comment?: string;
  is_anonymous: boolean;
  status: 'SUBMITTED' | 'ACKNOWLEDGED' | 'RESOLVED';
  acknowledged_at?: string;
  acknowledged_by?: string;
  staff_notes?: string;
  created_at: string;
  updated_at: string;
}

export interface SubmitFeedbackPayload {
  appointment_id: string;
  session_id?: string;
  clinic_id?: string;
  doctor_id?: string;
  therapist_id?: string;
  rating?: number;
  overall_rating?: number;
  therapist_rating?: number;
  facility_cleanliness_rating?: number;
  comment?: string;
  comments?: string;
  is_anonymous?: boolean;
}

export interface AdverseEventItem {
  id: string;
  clinic_id: string;
  patient_id: string;
  appointment_id?: string;
  session_id?: string;
  reported_by: string;
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL' | 'MILD' | 'MODERATE' | 'SEVERE';
  symptoms: string;
  vitals?: Record<string, any>;
  vitals_at_incident?: Record<string, any>;
  immediate_actions?: string;
  immediate_actions_taken?: string;
  status: 'REPORTED' | 'UNDER_REVIEW' | 'ACKNOWLEDGED' | 'RESOLVED' | 'CLOSED';
  acknowledged_at?: string;
  acknowledged_by?: string;
  staff_notes?: string;
  reported_at: string;
  resolved_at?: string;
}

export interface ReportAdverseEventPayload {
  clinic_id?: string;
  appointment_id?: string;
  session_id?: string;
  patient_id?: string;
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL' | 'MILD' | 'MODERATE' | 'SEVERE';
  symptoms: string;
  vitals?: Record<string, any>;
  vitals_at_incident?: Record<string, any>;
  immediate_actions?: string;
  immediate_actions_taken?: string;
}

export interface MetricValue {
  value: number;
  unit: string;
  description: string;
}

export interface DetailedClinicDashboard {
  clinic_id: string;
  period_start: string;
  period_end: string;
  booking_conversion: MetricValue;
  cancellation_rate: MetricValue;
  no_show_rate: MetricValue;
  checklist_adherence: MetricValue;
  avg_session_duration_minutes: MetricValue;
  avg_feedback_rating: MetricValue;
  therapist_utilization: MetricValue;
  follow_up_completion_rate: MetricValue;
  complications: {
    total_count: number;
    by_severity: {
      mild: number;
      moderate: number;
      severe: number;
      critical: number;
    };
  };
  daily_rollups: Array<{
    date: string;
    completed_sessions: number;
    cancelled_appointments: number;
    avg_rating: number;
    complications_count: number;
  }>;
}

export interface ClinicMetrics {
  clinic_id: string;
  total_appointments: number;
  confirmed_bookings: number;
  pending_checkins: number;
  sessions_in_progress: number;
  completed_sessions: number;
  cancellations: number;
  average_rating: number;
  room_utilization_pct: number;
  daily_trends?: Array<{
    date: string;
    completed: number;
    cancelled: number;
  }>;
}

