import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/sonner';
import { AuthProvider } from '@/context/AuthContext';
import { ProtectedRoute } from '@/components/layout/ProtectedRoute';

// Public Pages
import { LandingPage } from '@/pages/LandingPage';
import { LoginPage } from '@/pages/LoginPage';
import { RegisterPage } from '@/pages/RegisterPage';
import { ClinicsCatalogPage } from '@/pages/ClinicsCatalogPage';
import { ClinicDetailPage } from '@/pages/ClinicDetailPage';
import NotFoundPage from '@/pages/NotFound';

// Layouts
import { PatientLayout } from '@/components/layout/PatientLayout';
import { ClinicLayout } from '@/components/layout/ClinicLayout';
import { TherapistLayout } from '@/components/layout/TherapistLayout';

// Patient Portal Pages
import { PatientDashboard } from '@/pages/patient/PatientDashboard';
import { PatientBookAppointment } from '@/pages/patient/PatientBookAppointment';
import { PatientAppointmentsPage } from '@/pages/patient/PatientAppointmentsPage';
import { PatientChecklistsPage } from '@/pages/patient/PatientChecklistsPage';
import { PatientSessionProgressPage } from '@/pages/patient/PatientSessionProgressPage';
import { PatientProfilePage } from '@/pages/patient/PatientProfilePage';

// Clinic Admin Pages
import { ClinicOverviewDashboard } from '@/pages/clinic/ClinicOverviewDashboard';
import { ClinicAppointmentsPage } from '@/pages/clinic/ClinicAppointmentsPage';
import { ClinicDoctorsPage } from '@/pages/clinic/ClinicDoctorsPage';
import { ClinicRoomsPage } from '@/pages/clinic/ClinicRoomsPage';
import { ClinicPackagesPage } from '@/pages/clinic/ClinicPackagesPage';
import { ClinicReportsPage } from '@/pages/clinic/ClinicReportsPage';

// Therapist Workspace Pages
import { TherapistDashboard } from '@/pages/therapist/TherapistDashboard';
import { TherapistSessionExecution } from '@/pages/therapist/TherapistSessionExecution';
import { TherapistFollowUpsPage } from '@/pages/therapist/TherapistFollowUpsPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60 * 1000,
      retry: 1,
    },
  },
});

export const App: React.FC = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <Toaster />
          <Routes>
            {/* Public Routes */}
            <Route path="/" element={<LandingPage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/clinics" element={<ClinicsCatalogPage />} />
            <Route path="/clinics/:clinicId" element={<ClinicDetailPage />} />

            {/* Patient Portal Routes */}
            <Route
              path="/patient"
              element={
                <ProtectedRoute allowedRoles={['patient']}>
                  <PatientLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<Navigate to="/patient/dashboard" replace />} />
              <Route path="dashboard" element={<PatientDashboard />} />
              <Route path="book" element={<PatientBookAppointment />} />
              <Route path="appointments" element={<PatientAppointmentsPage />} />
              <Route path="checklists" element={<PatientChecklistsPage />} />
              <Route path="progress" element={<PatientSessionProgressPage />} />
              <Route path="profile" element={<PatientProfilePage />} />
            </Route>

            {/* Clinic Administrator Routes */}
            <Route
              path="/clinic"
              element={
                <ProtectedRoute allowedRoles={['clinic_admin', 'super_admin']}>
                  <ClinicLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<Navigate to="/clinic/dashboard" replace />} />
              <Route path="dashboard" element={<ClinicOverviewDashboard />} />
              <Route path="appointments" element={<ClinicAppointmentsPage />} />
              <Route path="doctors" element={<ClinicDoctorsPage />} />
              <Route path="rooms" element={<ClinicRoomsPage />} />
              <Route path="packages" element={<ClinicPackagesPage />} />
              <Route path="reports" element={<ClinicReportsPage />} />
            </Route>

            {/* Therapist Workspace Routes */}
            <Route
              path="/therapist"
              element={
                <ProtectedRoute allowedRoles={['therapist', 'doctor']}>
                  <TherapistLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<Navigate to="/therapist/dashboard" replace />} />
              <Route path="dashboard" element={<TherapistDashboard />} />
              <Route path="session-tracker" element={<TherapistSessionExecution />} />
              <Route path="follow-ups" element={<TherapistFollowUpsPage />} />
            </Route>

            {/* Fallback 404 */}
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
};

export default App;