import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Calendar, Clock, LogOut, Play, Pause, CheckCircle, User } from 'lucide-react';
import { User as UserType, Doctor, Appointment, Session } from '@/lib/types';
import { getStoredData, setStoredData } from '@/lib/data';
import SessionTracker from '@/components/SessionTracker';

interface TherapistDashboardProps {
  user: UserType;
  onLogout: () => void;
}

export default function TherapistDashboard({ user, onLogout }: TherapistDashboardProps) {
  const [doctor, setDoctor] = useState<Doctor | null>(null);
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [patients, setPatients] = useState<UserType[]>([]);
  const [selectedAppointment, setSelectedAppointment] = useState<Appointment | null>(null);

  useEffect(() => {
    // Find the doctor profile for this user
    const allDoctors = getStoredData('doctors', []);
    const userDoctor = allDoctors.find(d => d.name.toLowerCase().includes(user.name.toLowerCase().split(' ')[0]));
    setDoctor(userDoctor || null);

    if (userDoctor) {
      // Get appointments for this doctor - only show approved, scheduled, confirmed, and active appointments
      const allAppointments = getStoredData('appointments', []);
      const doctorAppointments = allAppointments.filter(apt => 
        apt.doctorId === userDoctor.id && 
        (apt.status === 'approved' || 
         apt.status === 'scheduled' || 
         apt.status === 'confirmed' || 
         apt.status === 'checked_in' || 
         apt.status === 'in_progress' || 
         apt.status === 'completed')
      );
      setAppointments(doctorAppointments);

      // Get sessions
      const allSessions = getStoredData('sessions', []);
      setSessions(allSessions);

      // Get patients
      const allUsers = getStoredData('users', []);
      const patientIds = [...new Set(doctorAppointments.map(apt => apt.patientId))];
      const appointmentPatients = allUsers.filter(u => patientIds.includes(u.id));
      setPatients(appointmentPatients);
    }
  }, [user]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'approved': return 'bg-green-100 text-green-800';
      case 'scheduled': return 'bg-blue-100 text-blue-800';
      case 'confirmed': return 'bg-green-100 text-green-800';
      case 'checked_in': return 'bg-yellow-100 text-yellow-800';
      case 'in_progress': return 'bg-purple-100 text-purple-800';
      case 'completed': return 'bg-gray-100 text-gray-800';
      case 'cancelled': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'approved': return 'APPROVED';
      case 'scheduled': return 'SCHEDULED';
      case 'confirmed': return 'CONFIRMED';
      case 'checked_in': return 'CHECKED IN';
      case 'in_progress': return 'IN PROGRESS';
      case 'completed': return 'COMPLETED';
      case 'cancelled': return 'CANCELLED';
      default: return status.toUpperCase();
    }
  };

  const getTodayAppointments = () => {
    const today = new Date().toISOString().split('T')[0];
    return appointments.filter(apt => apt.date === today);
  };

  const getUpcomingAppointments = () => {
    const today = new Date().toISOString().split('T')[0];
    return appointments.filter(apt => apt.date > today).slice(0, 10);
  };

  const getSessionForAppointment = (appointmentId: string) => {
    return sessions.find(s => s.appointmentId === appointmentId);
  };

  const handleConfirmAppointment = (appointment: Appointment) => {
    const allAppointments = getStoredData('appointments', []);
    const updatedAppointments = allAppointments.map(apt => 
      apt.id === appointment.id ? { ...apt, status: 'confirmed' as const } : apt
    );
    setStoredData('appointments', updatedAppointments);
    
    // Update local state
    setAppointments(updatedAppointments.filter(apt => 
      apt.doctorId === doctor?.id && 
      (apt.status === 'approved' || 
       apt.status === 'scheduled' || 
       apt.status === 'confirmed' || 
       apt.status === 'checked_in' || 
       apt.status === 'in_progress' || 
       apt.status === 'completed')
    ));
  };

  if (selectedAppointment) {
    return (
      <SessionTracker
        appointment={selectedAppointment}
        doctor={doctor!}
        onBack={() => setSelectedAppointment(null)}
        onSessionComplete={() => {
          setSelectedAppointment(null);
          // Refresh data
          const allSessions = getStoredData('sessions', []);
          setSessions(allSessions);
        }}
      />
    );
  }

  if (!doctor) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-600">Loading therapist information...</p>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Dr. {doctor.name}</h1>
              <p className="text-gray-600">Therapist Dashboard</p>
            </div>
            <Button variant="outline" onClick={onLogout}>
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-8">
        {/* Stats Cards */}
        <div className="grid md:grid-cols-4 gap-6 mb-8">
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Today's Sessions</p>
                  <p className="text-2xl font-bold text-gray-900">{getTodayAppointments().length}</p>
                </div>
                <Calendar className="h-8 w-8 text-blue-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Active Sessions</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {appointments.filter(apt => apt.status === 'in_progress').length}
                  </p>
                </div>
                <Play className="h-8 w-8 text-green-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Completed Today</p>
                  <p className="text-2xl font-bold text-gray-900">
                    {getTodayAppointments().filter(apt => apt.status === 'completed').length}
                  </p>
                </div>
                <CheckCircle className="h-8 w-8 text-purple-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Total Patients</p>
                  <p className="text-2xl font-bold text-gray-900">{patients.length}</p>
                </div>
                <User className="h-8 w-8 text-orange-600" />
              </div>
            </CardContent>
          </Card>
        </div>

        <Tabs defaultValue="today" className="space-y-6">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="today">Today's Schedule</TabsTrigger>
            <TabsTrigger value="upcoming">Upcoming</TabsTrigger>
            <TabsTrigger value="profile">My Profile</TabsTrigger>
          </TabsList>

          <TabsContent value="today" className="space-y-6">
            <div>
              <h2 className="text-xl font-semibold mb-4">
                Today's Sessions - {new Date().toLocaleDateString('en-US', { 
                  weekday: 'long', 
                  year: 'numeric', 
                  month: 'long', 
                  day: 'numeric' 
                })}
              </h2>
              
              {getTodayAppointments().length === 0 ? (
                <Card>
                  <CardContent className="text-center py-8">
                    <Calendar className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                    <p className="text-gray-600">No sessions scheduled for today.</p>
                    <p className="text-sm text-gray-500 mt-2">Enjoy your day off!</p>
                  </CardContent>
                </Card>
              ) : (
                <div className="space-y-4">
                  {getTodayAppointments()
                    .sort((a, b) => a.time.localeCompare(b.time))
                    .map((appointment) => {
                      const patient = patients.find(p => p.id === appointment.patientId);
                      const packages = getStoredData('packages', []);
                      const pkg = packages.find(p => p.id === appointment.packageId);
                      const session = getSessionForAppointment(appointment.id);

                      return (
                        <Card key={appointment.id} className="hover:shadow-md transition-shadow">
                          <CardHeader>
                            <div className="flex items-center justify-between">
                              <div className="flex items-center space-x-4">
                                <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center">
                                  <User className="h-6 w-6 text-blue-600" />
                                </div>
                                <div>
                                  <CardTitle className="text-lg">{patient?.name}</CardTitle>
                                  <CardDescription>{pkg?.name}</CardDescription>
                                </div>
                              </div>
                              <div className="text-right">
                                <div className="flex items-center text-lg font-semibold mb-1">
                                  <Clock className="h-4 w-4 mr-1" />
                                  {appointment.time}
                                </div>
                                <Badge className={getStatusColor(appointment.status)}>
                                  {getStatusText(appointment.status)}
                                </Badge>
                              </div>
                            </div>
                          </CardHeader>
                          <CardContent>
                            <div className="grid md:grid-cols-3 gap-4 mb-4">
                              <div>
                                <p className="text-sm text-gray-600">Duration</p>
                                <p className="font-medium">{pkg?.durationMinutes} minutes</p>
                              </div>
                              <div>
                                <p className="text-sm text-gray-600">Steps</p>
                                <p className="font-medium">{pkg?.steps.length} procedures</p>
                              </div>
                              <div>
                                <p className="text-sm text-gray-600">Contact</p>
                                <p className="font-medium">{patient?.phone}</p>
                              </div>
                            </div>

                            {appointment.notes && (
                              <div className="mb-4 p-3 bg-yellow-50 rounded-lg">
                                <p className="text-sm text-yellow-800">
                                  <strong>Patient Notes:</strong> {appointment.notes}
                                </p>
                              </div>
                            )}

                            {appointment.status === 'approved' && (
                              <div className="mb-4 p-3 bg-green-50 rounded-lg">
                                <p className="text-sm text-green-800">
                                  <strong>New Assignment:</strong> This appointment has been approved by the clinic admin and assigned to you.
                                </p>
                              </div>
                            )}

                            {session && (
                              <div className="mb-4 p-3 bg-blue-50 rounded-lg">
                                <p className="text-sm text-blue-800">
                                  <strong>Session Progress:</strong> {session.completedSteps.length} of {pkg?.steps.length} steps completed
                                </p>
                              </div>
                            )}

                            <div className="flex gap-2">
                              {appointment.status === 'approved' && (
                                <Button 
                                  variant="outline" 
                                  size="sm"
                                  onClick={() => handleConfirmAppointment(appointment)}
                                >
                                  Confirm Appointment
                                </Button>
                              )}

                              {(appointment.status === 'scheduled' || appointment.status === 'confirmed') && (
                                <Button 
                                  variant="outline" 
                                  size="sm"
                                  onClick={() => {
                                    // Update status to checked_in
                                    const allAppointments = getStoredData('appointments', []);
                                    const updatedAppointments = allAppointments.map(apt => 
                                      apt.id === appointment.id ? { ...apt, status: 'checked_in' as const } : apt
                                    );
                                    setStoredData('appointments', updatedAppointments);
                                    setAppointments(updatedAppointments.filter(apt => 
                                      apt.doctorId === doctor.id && 
                                      (apt.status === 'approved' || 
                                       apt.status === 'scheduled' || 
                                       apt.status === 'confirmed' || 
                                       apt.status === 'checked_in' || 
                                       apt.status === 'in_progress' || 
                                       apt.status === 'completed')
                                    ));
                                  }}
                                >
                                  Check In Patient
                                </Button>
                              )}
                              
                              {(appointment.status === 'checked_in' || appointment.status === 'in_progress') && (
                                <Button 
                                  size="sm"
                                  onClick={() => setSelectedAppointment(appointment)}
                                >
                                  {appointment.status === 'checked_in' ? 'Start Session' : 'Continue Session'}
                                </Button>
                              )}

                              {appointment.status === 'completed' && (
                                <Button 
                                  variant="outline" 
                                  size="sm"
                                  onClick={() => setSelectedAppointment(appointment)}
                                >
                                  View Session Details
                                </Button>
                              )}

                              <Button variant="ghost" size="sm">
                                Contact Patient
                              </Button>
                            </div>
                          </CardContent>
                        </Card>
                      );
                    })}
                </div>
              )}
            </div>
          </TabsContent>

          <TabsContent value="upcoming" className="space-y-6">
            <div>
              <h2 className="text-xl font-semibold mb-4">Upcoming Appointments</h2>
              
              {getUpcomingAppointments().length === 0 ? (
                <Card>
                  <CardContent className="text-center py-8">
                    <Calendar className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                    <p className="text-gray-600">No upcoming appointments scheduled.</p>
                  </CardContent>
                </Card>
              ) : (
                <div className="space-y-4">
                  {getUpcomingAppointments().map((appointment) => {
                    const patient = patients.find(p => p.id === appointment.patientId);
                    const packages = getStoredData('packages', []);
                    const pkg = packages.find(p => p.id === appointment.packageId);

                    return (
                      <Card key={appointment.id}>
                        <CardContent className="p-6">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-4">
                              <div className="w-10 h-10 bg-gray-100 rounded-full flex items-center justify-center">
                                <User className="h-5 w-5 text-gray-600" />
                              </div>
                              <div>
                                <p className="font-medium text-gray-900">{patient?.name}</p>
                                <p className="text-sm text-gray-600">{pkg?.name}</p>
                                <Badge className={getStatusColor(appointment.status)} size="sm">
                                  {getStatusText(appointment.status)}
                                </Badge>
                              </div>
                            </div>
                            <div className="text-right">
                              <p className="font-medium">
                                {new Date(appointment.date).toLocaleDateString()} at {appointment.time}
                              </p>
                              <p className="text-sm text-gray-600">{pkg?.durationMinutes} minutes</p>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    );
                  })}
                </div>
              )}
            </div>
          </TabsContent>

          <TabsContent value="profile" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Therapist Profile</CardTitle>
                <CardDescription>Your professional information and statistics</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="flex items-start space-x-6">
                  <img 
                    src={doctor.image} 
                    alt={doctor.name}
                    className="w-24 h-24 rounded-full object-cover"
                  />
                  <div className="flex-1 space-y-4">
                    <div className="grid md:grid-cols-2 gap-4">
                      <div>
                        <label className="text-sm font-medium text-gray-700">Name</label>
                        <p className="text-gray-900">{doctor.name}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-700">Qualifications</label>
                        <p className="text-gray-900">{doctor.qualifications}</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-700">Experience</label>
                        <p className="text-gray-900">{doctor.experience} years</p>
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-700">Rating</label>
                        <p className="text-gray-900">{doctor.rating}/5.0</p>
                      </div>
                    </div>

                    <div>
                      <label className="text-sm font-medium text-gray-700">Specialties</label>
                      <div className="flex flex-wrap gap-2 mt-1">
                        {doctor.specialties.map((specialty) => (
                          <Badge key={specialty} variant="secondary">
                            {specialty}
                          </Badge>
                        ))}
                      </div>
                    </div>

                    <div>
                      <label className="text-sm font-medium text-gray-700">Weekly Schedule</label>
                      <div className="mt-2 space-y-1">
                        {Object.entries(doctor.schedule).map(([day, slots]) => (
                          <div key={day} className="flex justify-between text-sm">
                            <span className="capitalize font-medium">{day}:</span>
                            <span className="text-gray-600">
                              {slots.length > 0 ? `${slots[0].start} - ${slots[0].end}` : 'Off'}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>

                <div className="grid md:grid-cols-3 gap-4 pt-6 border-t">
                  <div className="text-center">
                    <p className="text-2xl font-bold text-blue-600">{appointments.length}</p>
                    <p className="text-sm text-gray-600">Total Appointments</p>
                  </div>
                  <div className="text-center">
                    <p className="text-2xl font-bold text-green-600">
                      {appointments.filter(apt => apt.status === 'completed').length}
                    </p>
                    <p className="text-sm text-gray-600">Completed Sessions</p>
                  </div>
                  <div className="text-center">
                    <p className="text-2xl font-bold text-purple-600">{patients.length}</p>
                    <p className="text-sm text-gray-600">Unique Patients</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}