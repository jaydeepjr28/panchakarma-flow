export type UserRole = 'patient' | 'clinic_admin' | 'therapist';

export interface User {
  id: string;
  name: string;
  email: string;
  phone: string;
  role: UserRole;
  clinicId?: string; // For clinic_admin and therapist
}

export interface Clinic {
  id: string;
  name: string;
  address: string;
  phone: string;
  email: string;
  hours: string;
  languages: string[];
  description: string;
  rating: number;
  image: string;
}

export interface Doctor {
  id: string;
  clinicId: string;
  name: string;
  specialties: string[];
  qualifications: string;
  experience: number;
  schedule: WeeklySchedule;
  maxSessionsPerDay: number;
  rating: number;
  image: string;
}

export interface WeeklySchedule {
  [key: string]: TimeSlot[]; // monday, tuesday, etc.
}

export interface TimeSlot {
  start: string; // "09:00"
  end: string;   // "17:00"
}

export interface TherapyPackage {
  id: string;
  clinicId: string;
  name: string;
  description: string;
  steps: TherapyStep[];
  totalSessions: number;
  durationMinutes: number;
  price: number;
  image: string;
}

export interface TherapyStep {
  id: string;
  name: string;
  description: string;
  durationMinutes: number;
  instructions: string[];
}

export type AppointmentStatus = 'pending' | 'approved' | 'scheduled' | 'confirmed' | 'checked_in' | 'in_progress' | 'completed' | 'cancelled' | 'rejected';

export interface Appointment {
  id: string;
  patientId: string;
  clinicId: string;
  doctorId: string;
  packageId: string;
  date: string; // YYYY-MM-DD
  time: string; // HH:MM
  status: AppointmentStatus;
  notes?: string;
  createdAt: string;
  approvedAt?: string;
  approvedBy?: string; // clinic admin user id
  rejectionReason?: string;
}

export interface Session {
  id: string;
  appointmentId: string;
  currentStep: number;
  completedSteps: SessionStep[];
  startTime?: string;
  endTime?: string;
  therapistNotes: string;
  patientFeedback?: string;
  rating?: number;
}

export interface SessionStep {
  stepId: string;
  completed: boolean;
  startTime?: string;
  endTime?: string;
  notes: string;
}

export interface PreProcedureChecklist {
  id: string;
  appointmentId: string;
  items: ChecklistItem[];
  completedAt?: string;
}

export interface ChecklistItem {
  id: string;
  description: string;
  completed: boolean;
  completedAt?: string;
}

export interface Notification {
  id: string;
  userId: string;
  title: string;
  message: string;
  type: 'info' | 'warning' | 'success' | 'error';
  read: boolean;
  createdAt: string;
}