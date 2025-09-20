import { User, Clinic, Doctor, TherapyPackage, Appointment, Session, Notification, PreProcedureChecklist } from './types';

// Mock data for MVP
export const mockUsers: User[] = [
  {
    id: 'user1',
    name: 'John Patient',
    email: 'john@example.com',
    phone: '+1234567890',
    role: 'patient'
  },
  {
    id: 'user2',
    name: 'Dr. Sarah Admin',
    email: 'sarah@ayurclinic.com',
    phone: '+1234567891',
    role: 'clinic_admin',
    clinicId: 'clinic1'
  },
  {
    id: 'user3',
    name: 'Dr. Raj Therapist',
    email: 'raj@ayurclinic.com',
    phone: '+1234567892',
    role: 'therapist',
    clinicId: 'clinic1'
  }
];

export const mockClinics: Clinic[] = [
  {
    id: 'clinic1',
    name: 'Ayur Wellness Center',
    address: '123 Wellness Street, Health City, HC 12345',
    phone: '+1234567890',
    email: 'info@ayurwellness.com',
    hours: 'Mon-Sat: 8:00 AM - 6:00 PM',
    languages: ['English', 'Hindi', 'Sanskrit'],
    description: 'Premier Panchakarma therapy center with experienced practitioners and traditional methods.',
    rating: 4.8,
    image: 'https://images.unsplash.com/photo-1559757148-5c350d0d3c56?w=400'
  },
  {
    id: 'clinic2',
    name: 'Traditional Ayurveda Clinic',
    address: '456 Herbal Avenue, Wellness Town, WT 67890',
    phone: '+1234567893',
    email: 'contact@traditionalayur.com',
    hours: 'Mon-Fri: 9:00 AM - 5:00 PM',
    languages: ['English', 'Tamil', 'Malayalam'],
    description: 'Authentic Ayurvedic treatments following ancient traditions with modern facilities.',
    rating: 4.6,
    image: 'https://images.unsplash.com/photo-1544161515-4ab6ce6db874?w=400'
  }
];

export const mockDoctors: Doctor[] = [
  {
    id: 'doctor1',
    clinicId: 'clinic1',
    name: 'Dr. Raj Kumar',
    specialties: ['Panchakarma', 'Abhyanga', 'Svedana'],
    qualifications: 'BAMS, MD (Ayurveda)',
    experience: 15,
    schedule: {
      sunday: [],
      monday: [{ start: '09:00', end: '17:00' }],
      tuesday: [{ start: '09:00', end: '17:00' }],
      wednesday: [{ start: '09:00', end: '17:00' }],
      thursday: [{ start: '09:00', end: '17:00' }],
      friday: [{ start: '09:00', end: '17:00' }],
      saturday: [{ start: '09:00', end: '13:00' }]
    },
    maxSessionsPerDay: 8,
    rating: 4.9,
    image: 'https://images.unsplash.com/photo-1612349317150-e413f6a5b16d?w=200'
  },
  {
    id: 'doctor2',
    clinicId: 'clinic1',
    name: 'Dr. Priya Sharma',
    specialties: ['Panchakarma', 'Nasya', 'Basti'],
    qualifications: 'BAMS, PhD (Panchakarma)',
    experience: 12,
    schedule: {
      sunday: [],
      monday: [{ start: '10:00', end: '18:00' }],
      tuesday: [{ start: '10:00', end: '18:00' }],
      wednesday: [{ start: '10:00', end: '18:00' }],
      thursday: [{ start: '10:00', end: '18:00' }],
      friday: [{ start: '10:00', end: '18:00' }],
      saturday: []
    },
    maxSessionsPerDay: 6,
    rating: 4.7,
    image: 'https://images.unsplash.com/photo-1559839734-2b71ea197ec2?w=200'
  }
];

export const mockPackages: TherapyPackage[] = [
  {
    id: 'package1',
    clinicId: 'clinic1',
    name: 'Complete Panchakarma Detox',
    description: 'A comprehensive 7-day Panchakarma program including all five procedures for complete detoxification.',
    steps: [
      {
        id: 'step1',
        name: 'Abhyanga (Oil Massage)',
        description: 'Full body massage with medicated oils',
        durationMinutes: 45,
        instructions: ['Arrive 15 minutes early', 'Wear comfortable clothing', 'Avoid heavy meals 2 hours before']
      },
      {
        id: 'step2',
        name: 'Svedana (Steam Therapy)',
        description: 'Herbal steam therapy for opening channels',
        durationMinutes: 20,
        instructions: ['Stay hydrated', 'Inform if you feel dizzy', 'Relax and breathe deeply']
      },
      {
        id: 'step3',
        name: 'Virechana (Purgation)',
        description: 'Controlled elimination therapy',
        durationMinutes: 30,
        instructions: ['Follow fasting guidelines', 'Stay near restroom facilities', 'Drink plenty of water']
      }
    ],
    totalSessions: 7,
    durationMinutes: 95,
    price: 2500,
    image: 'https://images.unsplash.com/photo-1544161515-4ab6ce6db874?w=300'
  },
  {
    id: 'package2',
    clinicId: 'clinic1',
    name: 'Rejuvenation Therapy',
    description: 'A gentle 3-day program focused on relaxation and rejuvenation.',
    steps: [
      {
        id: 'step1',
        name: 'Abhyanga (Oil Massage)',
        description: 'Relaxing full body massage',
        durationMinutes: 60,
        instructions: ['Arrive relaxed', 'Wear loose clothing', 'Avoid caffeine before treatment']
      },
      {
        id: 'step2',
        name: 'Shirodhara (Oil Pouring)',
        description: 'Continuous pouring of medicated oil on forehead',
        durationMinutes: 30,
        instructions: ['Keep eyes closed', 'Breathe naturally', 'Let your mind relax']
      }
    ],
    totalSessions: 3,
    durationMinutes: 90,
    price: 1200,
    image: 'https://images.unsplash.com/photo-1540555700478-4be289fbecef?w=300'
  }
];

// Data management functions
export const getStoredData = <T>(key: string, defaultValue: T[]): T[] => {
  try {
    const stored = localStorage.getItem(key);
    return stored ? JSON.parse(stored) : defaultValue;
  } catch {
    return defaultValue;
  }
};

export const setStoredData = <T>(key: string, data: T[]): void => {
  localStorage.setItem(key, JSON.stringify(data));
};

// Initialize data if not exists
export const initializeData = () => {
  if (!localStorage.getItem('users')) {
    setStoredData('users', mockUsers);
  }
  if (!localStorage.getItem('clinics')) {
    setStoredData('clinics', mockClinics);
  }
  if (!localStorage.getItem('doctors')) {
    setStoredData('doctors', mockDoctors);
  }
  if (!localStorage.getItem('packages')) {
    setStoredData('packages', mockPackages);
  }
  if (!localStorage.getItem('appointments')) {
    setStoredData('appointments', []);
  }
  if (!localStorage.getItem('sessions')) {
    setStoredData('sessions', []);
  }
  if (!localStorage.getItem('notifications')) {
    setStoredData('notifications', []);
  }
};

// Utility functions
export const getCurrentUser = (): User | null => {
  const userId = localStorage.getItem('currentUserId');
  if (!userId) return null;
  
  const users = getStoredData('users', mockUsers);
  return users.find(user => user.id === userId) || null;
};

export const setCurrentUser = (userId: string) => {
  localStorage.setItem('currentUserId', userId);
};

export const logout = () => {
  localStorage.removeItem('currentUserId');
};

export const generateId = () => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9);
};

export const formatDate = (date: Date) => {
  return date.toISOString().split('T')[0];
};

export const formatTime = (date: Date) => {
  return date.toTimeString().slice(0, 5);
};