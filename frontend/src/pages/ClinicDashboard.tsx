import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Calendar, Users, Package, LogOut, Clock, Star, Phone, AlertCircle, CheckCircle, XCircle } from 'lucide-react';
import { User, Clinic, Doctor, TherapyPackage, Appointment } from '@/lib/types';
import { getStoredData, setStoredData, generateId } from '@/lib/data';

interface ClinicDashboardProps {
  user: User;
  onLogout: () => void;
}

export default function ClinicDashboard({ user, onLogout }: ClinicDashboardProps) {
  const [clinic, setClinic] = useState<Clinic | null>(null);
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [packages, setPackages] = useState<TherapyPackage[]>([]);
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [patients, setPatients] = useState<User[]>([]);
  const [rejectionReason, setRejectionReason] = useState<string>('');
  const [selectedAppointment, setSelectedAppointment] = useState<Appointment | null>(null);

  useEffect(() => {
    if (user.clinicId) {
      const clinics = getStoredData('clinics', []);
      const foundClinic = clinics.find(c => c.id === user.clinicId);
      setClinic(foundClinic || null);

      const allDoctors = getStoredData('doctors', []);
      setDoctors(allDoctors.filter(d => d.clinicId === user.clinicId));

      const allPackages = getStoredData('packages', []);
      setPackages(allPackages.filter(p => p.clinicId === user.clinicId));

      const allAppointments = getStoredData('appointments', []);
      const clinicAppointments = allAppointments.filter(apt => apt.clinicId === user.clinicId);
      setAppointments(clinicAppointments);

      // Get unique patients
      const allUsers = getStoredData('users', []);
      const patientIds = [...new Set(clinicAppointments.map(apt => apt.patientId))];
      const clinicPatients = allUsers.filter(u => patientIds.includes(u.id));
      setPatients(clinicPatients);
    }
  }, [user.clinicId]);

  const refreshAppointments = () => {
    const allAppointments = getStoredData('appointments', []);
    const clinicAppointments = allAppointments.filter(apt => apt.clinicId === user.clinicId);
    setAppointments(clinicAppointments);
  };

  const handleApproveAppointment = (appointment: Appointment) => {
    const allAppointments = getStoredData('appointments', []);
    const updatedAppointments = allAppointments.map(apt => 
      apt.id === appointment.id 
        ? { 
            ...apt, 
            status: 'approved' as const, 
            approvedAt: new Date().toISOString(),
            approvedBy: user.id
          } 
        : apt
    );
    setStoredData('appointments', updatedAppointments);

    // Create notifications
    const notifications = getStoredData('notifications', []);
    
    // Notification for patient
    notifications.push({
      id: generateId(),
      userId: appointment.patientId,
      title: 'Appointment Approved!',
      message: `Your appointment request at ${clinic?.name} for ${new Date(appointment.date).toLocaleDateString()} at ${appointment.time} has been approved. You will receive final confirmation details soon.`,
      type: 'success',
      read: false,
      createdAt: new Date().toISOString()
    });

    // Notification for assigned therapist
    const therapist = getStoredData('users', []).find(u => 
      u.clinicId === user.clinicId && 
      u.role === 'therapist'
    );
    
    if (therapist) {
      const patient = patients.find(p => p.id === appointment.patientId);
      const pkg = packages.find(p => p.id === appointment.packageId);
      
      notifications.push({
        id: generateId(),
        userId: therapist.id,
        title: 'New Appointment Assigned',
        message: `New appointment: ${patient?.name} - ${pkg?.name} on ${new Date(appointment.date).toLocaleDateString()} at ${appointment.time}`,
        type: 'info',
        read: false,
        createdAt: new Date().toISOString()
      });
    }

    setStoredData('notifications', notifications);
    refreshAppointments();
  };

  const handleRejectAppointment = (appointment: Appointment, reason: string) => {
    const allAppointments = getStoredData('appointments', []);
    const updatedAppointments = allAppointments.map(apt => 
      apt.id === appointment.id 
        ? { 
            ...apt, 
            status: 'rejected' as const, 
            rejectionReason: reason,
            approvedBy: user.id
          } 
        : apt
    );
    setStoredData('appointments', updatedAppointments);

    // Create notification for patient
    const notifications = getStoredData('notifications', []);
    notifications.push({
      id: generateId(),
      userId: appointment.patientId,
      title: 'Appointment Request Update',
      message: `Your appointment request at ${clinic?.name} for ${new Date(appointment.date).toLocaleDateString()} could not be approved. Reason: ${reason}`,
      type: 'warning',
      read: false,
      createdAt: new Date().toISOString()
    });

    setStoredData('notifications', notifications);
    refreshAppointments();
    setRejectionReason('');
    setSelectedAppointment(null);
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pending': return 'bg-yellow-100 text-yellow-800';
      case 'approved': return 'bg-green-100 text-green-800';
      case 'scheduled': return 'bg-blue-100 text-blue-800';
      case 'confirmed': return 'bg-green-100 text-green-800';
      case 'checked_in': return 'bg-yellow-100 text-yellow-800';
      case 'in_progress': return 'bg-purple-100 text-purple-800';
      case 'completed': return 'bg-gray-100 text-gray-800';
      case 'cancelled': return 'bg-red-100 text-red-800';
      case 'rejected': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getPendingAppointments = () => {
    return appointments.filter(apt => apt.status === 'pending');
  };

  const getTodayAppointments = () => {
    const today = new Date().toISOString().split('T')[0];
    return appointments.filter(apt => 
      apt.date === today && 
      (apt.status === 'approved' || apt.status === 'scheduled' || apt.status === 'confirmed' || apt.status === 'checked_in' || apt.status === 'in_progress')
    );
  };

  const getUpcomingAppointments = () => {
    const today = new Date().toISOString().split('T')[0];
    return appointments.filter(apt => 
      apt.date > today && 
      (apt.status === 'approved' || apt.status === 'scheduled' || apt.status === 'confirmed')
    ).slice(0, 5);
  };

  if (!clinic) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-600">Loading clinic information...</p>
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
              <h1 className="text-2xl font-bold text-gray-900">{clinic.name}</h1>
              <p className="text-gray-600">Clinic Administration Dashboard</p>
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
        <div className="grid md:grid-cols-5 gap-6 mb-8">
          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Pending Requests</p>
                  <p className="text-2xl font-bold text-yellow-600">{getPendingAppointments().length}</p>
                </div>
                <AlertCircle className="h-8 w-8 text-yellow-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Today's Appointments</p>
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
                  <p className="text-sm font-medium text-gray-600">Total Doctors</p>
                  <p className="text-2xl font-bold text-gray-900">{doctors.length}</p>
                </div>
                <Users className="h-8 w-8 text-green-600" />
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-gray-600">Active Packages</p>
                  <p className="text-2xl font-bold text-gray-900">{packages.length}</p>
                </div>
                <Package className="h-8 w-8 text-purple-600" />
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
                <Users className="h-8 w-8 text-orange-600" />
              </div>
            </CardContent>
          </Card>
        </div>

        <Tabs defaultValue="pending" className="space-y-6">
          <TabsList className="grid w-full grid-cols-5">
            <TabsTrigger value="pending">Pending Requests</TabsTrigger>
            <TabsTrigger value="appointments">Appointments</TabsTrigger>
            <TabsTrigger value="doctors">Doctors</TabsTrigger>
            <TabsTrigger value="packages">Packages</TabsTrigger>
            <TabsTrigger value="clinic">Clinic Info</TabsTrigger>
          </TabsList>

          <TabsContent value="pending" className="space-y-6">
            <div>
              <h2 className="text-xl font-semibold mb-4">Pending Appointment Requests</h2>
              {getPendingAppointments().length === 0 ? (
                <Card>
                  <CardContent className="text-center py-8">
                    <CheckCircle className="h-12 w-12 text-green-400 mx-auto mb-4" />
                    <p className="text-gray-600">No pending requests.</p>
                    <p className="text-sm text-gray-500 mt-2">All appointment requests have been processed.</p>
                  </CardContent>
                </Card>
              ) : (
                <div className="space-y-4">
                  {getPendingAppointments()
                    .sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime())
                    .map((appointment) => {
                      const patient = patients.find(p => p.id === appointment.patientId);
                      const doctor = doctors.find(d => d.id === appointment.doctorId);
                      const pkg = packages.find(p => p.id === appointment.packageId);

                      return (
                        <Card key={appointment.id} className="border-l-4 border-l-yellow-400">
                          <CardHeader>
                            <div className="flex items-start justify-between">
                              <div>
                                <CardTitle className="text-lg">{patient?.name}</CardTitle>
                                <CardDescription>
                                  {pkg?.name} with {doctor?.name || 'Auto-assigned doctor'}
                                </CardDescription>
                              </div>
                              <Badge className={getStatusColor(appointment.status)}>
                                PENDING APPROVAL
                              </Badge>
                            </div>
                          </CardHeader>
                          <CardContent>
                            <div className="grid md:grid-cols-3 gap-4 mb-4">
                              <div>
                                <p className="text-sm text-gray-600">Date & Time</p>
                                <p className="font-medium">
                                  {new Date(appointment.date).toLocaleDateString()} at {appointment.time}
                                </p>
                              </div>
                              <div>
                                <p className="text-sm text-gray-600">Duration</p>
                                <p className="font-medium">{pkg?.durationMinutes} minutes</p>
                              </div>
                              <div>
                                <p className="text-sm text-gray-600">Price</p>
                                <p className="font-medium">₹{pkg?.price}</p>
                              </div>
                            </div>

                            <div className="grid md:grid-cols-2 gap-4 mb-4">
                              <div>
                                <p className="text-sm text-gray-600">Patient Contact</p>
                                <p className="font-medium">{patient?.phone}</p>
                                <p className="text-sm text-gray-500">{patient?.email}</p>
                              </div>
                              <div>
                                <p className="text-sm text-gray-600">Requested</p>
                                <p className="font-medium">
                                  {new Date(appointment.createdAt).toLocaleDateString()} at{' '}
                                  {new Date(appointment.createdAt).toLocaleTimeString()}
                                </p>
                              </div>
                            </div>

                            {appointment.notes && (
                              <div className="mb-4 p-3 bg-gray-50 rounded-lg">
                                <p className="text-sm text-gray-700">
                                  <strong>Patient Notes:</strong> {appointment.notes}
                                </p>
                              </div>
                            )}

                            <div className="flex gap-3">
                              <Button 
                                onClick={() => handleApproveAppointment(appointment)}
                                className="flex-1"
                              >
                                <CheckCircle className="h-4 w-4 mr-2" />
                                Approve Request
                              </Button>
                              
                              <Dialog>
                                <DialogTrigger asChild>
                                  <Button 
                                    variant="outline" 
                                    className="flex-1"
                                    onClick={() => setSelectedAppointment(appointment)}
                                  >
                                    <XCircle className="h-4 w-4 mr-2" />
                                    Reject Request
                                  </Button>
                                </DialogTrigger>
                                <DialogContent>
                                  <DialogHeader>
                                    <DialogTitle>Reject Appointment Request</DialogTitle>
                                    <DialogDescription>
                                      Please provide a reason for rejecting this appointment request. This will be shared with the patient.
                                    </DialogDescription>
                                  </DialogHeader>
                                  <div className="space-y-4">
                                    <div>
                                      <p className="text-sm font-medium mb-2">Patient: {patient?.name}</p>
                                      <p className="text-sm text-gray-600">
                                        {pkg?.name} on {new Date(appointment.date).toLocaleDateString()} at {appointment.time}
                                      </p>
                                    </div>
                                    <Textarea
                                      placeholder="Enter rejection reason (e.g., Doctor not available, Time slot conflict, etc.)"
                                      value={rejectionReason}
                                      onChange={(e) => setRejectionReason(e.target.value)}
                                      rows={3}
                                    />
                                  </div>
                                  <DialogFooter>
                                    <Button variant="outline" onClick={() => {
                                      setRejectionReason('');
                                      setSelectedAppointment(null);
                                    }}>
                                      Cancel
                                    </Button>
                                    <Button 
                                      variant="destructive"
                                      onClick={() => {
                                        if (selectedAppointment && rejectionReason.trim()) {
                                          handleRejectAppointment(selectedAppointment, rejectionReason.trim());
                                        }
                                      }}
                                      disabled={!rejectionReason.trim()}
                                    >
                                      Reject Request
                                    </Button>
                                  </DialogFooter>
                                </DialogContent>
                              </Dialog>
                            </div>
                          </CardContent>
                        </Card>
                      );
                    })}
                </div>
              )}
            </div>
          </TabsContent>

          <TabsContent value="appointments" className="space-y-6">
            <div className="grid lg:grid-cols-2 gap-6">
              {/* Today's Appointments */}
              <Card>
                <CardHeader>
                  <CardTitle>Today's Appointments</CardTitle>
                  <CardDescription>
                    {new Date().toLocaleDateString('en-US', { 
                      weekday: 'long', 
                      year: 'numeric', 
                      month: 'long', 
                      day: 'numeric' 
                    })}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {getTodayAppointments().length === 0 ? (
                    <p className="text-gray-600 text-center py-4">No appointments today</p>
                  ) : (
                    <div className="space-y-3">
                      {getTodayAppointments().map((appointment) => {
                        const doctor = doctors.find(d => d.id === appointment.doctorId);
                        const patient = patients.find(p => p.id === appointment.patientId);
                        const pkg = packages.find(p => p.id === appointment.packageId);

                        return (
                          <div key={appointment.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                            <div className="flex-1">
                              <p className="font-medium text-gray-900">{patient?.name}</p>
                              <p className="text-sm text-gray-600">{pkg?.name}</p>
                              <p className="text-sm text-gray-600">Dr. {doctor?.name}</p>
                            </div>
                            <div className="text-right">
                              <p className="font-medium">{appointment.time}</p>
                              <Badge className={getStatusColor(appointment.status)}>
                                {appointment.status.replace('_', ' ').toUpperCase()}
                              </Badge>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </CardContent>
              </Card>

              {/* Upcoming Appointments */}
              <Card>
                <CardHeader>
                  <CardTitle>Upcoming Appointments</CardTitle>
                  <CardDescription>Next 5 scheduled appointments</CardDescription>
                </CardHeader>
                <CardContent>
                  {getUpcomingAppointments().length === 0 ? (
                    <p className="text-gray-600 text-center py-4">No upcoming appointments</p>
                  ) : (
                    <div className="space-y-3">
                      {getUpcomingAppointments().map((appointment) => {
                        const doctor = doctors.find(d => d.id === appointment.doctorId);
                        const patient = patients.find(p => p.id === appointment.patientId);
                        const pkg = packages.find(p => p.id === appointment.packageId);

                        return (
                          <div key={appointment.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                            <div className="flex-1">
                              <p className="font-medium text-gray-900">{patient?.name}</p>
                              <p className="text-sm text-gray-600">{pkg?.name}</p>
                              <p className="text-sm text-gray-600">Dr. {doctor?.name}</p>
                            </div>
                            <div className="text-right">
                              <p className="text-sm text-gray-600">
                                {new Date(appointment.date).toLocaleDateString()}
                              </p>
                              <p className="font-medium">{appointment.time}</p>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          <TabsContent value="doctors" className="space-y-6">
            <div className="flex items-center justify-between">
              <h2 className="text-xl font-semibold">Doctors & Therapists</h2>
              <Button>Add New Doctor</Button>
            </div>

            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {doctors.map((doctor) => (
                <Card key={doctor.id}>
                  <CardHeader>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <CardTitle className="flex items-center">
                          {doctor.name}
                          <div className="flex items-center ml-2">
                            <Star className="h-4 w-4 text-yellow-500 fill-current" />
                            <span className="text-sm text-gray-600 ml-1">{doctor.rating}</span>
                          </div>
                        </CardTitle>
                        <CardDescription>{doctor.qualifications}</CardDescription>
                      </div>
                      <img 
                        src={doctor.image} 
                        alt={doctor.name}
                        className="w-12 h-12 rounded-full object-cover ml-4"
                      />
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      <div>
                        <span className="text-sm text-gray-600">Experience: </span>
                        <span className="font-medium">{doctor.experience} years</span>
                      </div>
                      
                      <div>
                        <span className="text-sm text-gray-600">Max Sessions/Day: </span>
                        <span className="font-medium">{doctor.maxSessionsPerDay}</span>
                      </div>

                      <div>
                        <span className="text-sm text-gray-600 block mb-1">Specialties:</span>
                        <div className="flex flex-wrap gap-1">
                          {doctor.specialties.map((specialty) => (
                            <Badge key={specialty} variant="secondary" className="text-xs">
                              {specialty}
                            </Badge>
                          ))}
                        </div>
                      </div>

                      <div>
                        <span className="text-sm text-gray-600 block mb-1">Schedule:</span>
                        <div className="text-xs text-gray-600">
                          {Object.entries(doctor.schedule).map(([day, slots]) => (
                            slots.length > 0 && (
                              <div key={day}>
                                {day}: {slots[0].start} - {slots[0].end}
                              </div>
                            )
                          ))}
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </TabsContent>

          <TabsContent value="packages" className="space-y-6">
            <div className="flex items-center justify-between">
              <h2 className="text-xl font-semibold">Therapy Packages</h2>
              <Button>Add New Package</Button>
            </div>

            <div className="grid md:grid-cols-2 gap-6">
              {packages.map((pkg) => (
                <Card key={pkg.id}>
                  <CardHeader>
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <CardTitle>{pkg.name}</CardTitle>
                        <CardDescription className="mt-2">{pkg.description}</CardDescription>
                      </div>
                      <img 
                        src={pkg.image} 
                        alt={pkg.name}
                        className="w-16 h-16 rounded-lg object-cover ml-4"
                      />
                    </div>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">Duration</span>
                        <span className="font-medium">{pkg.durationMinutes} minutes</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">Sessions</span>
                        <span className="font-medium">{pkg.totalSessions}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-600">Price</span>
                        <span className="font-bold text-lg text-green-600">₹{pkg.price}</span>
                      </div>
                    </div>
                    
                    <div className="mt-4">
                      <p className="text-sm font-medium mb-2">Treatment Steps:</p>
                      <div className="space-y-1">
                        {pkg.steps.map((step, index) => (
                          <div key={step.id} className="text-xs text-gray-600">
                            {index + 1}. {step.name} ({step.durationMinutes}min)
                          </div>
                        ))}
                      </div>
                    </div>

                    <div className="mt-4 flex gap-2">
                      <Button variant="outline" size="sm" className="flex-1">Edit</Button>
                      <Button variant="outline" size="sm" className="flex-1">View Details</Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </TabsContent>

          <TabsContent value="clinic" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Clinic Information</CardTitle>
                <CardDescription>Manage your clinic details and settings</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid md:grid-cols-2 gap-6">
                  <div className="space-y-4">
                    <div>
                      <label className="text-sm font-medium text-gray-700">Clinic Name</label>
                      <p className="text-gray-900">{clinic.name}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-gray-700">Address</label>
                      <p className="text-gray-900">{clinic.address}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-gray-700">Operating Hours</label>
                      <p className="text-gray-900">{clinic.hours}</p>
                    </div>
                  </div>
                  
                  <div className="space-y-4">
                    <div>
                      <label className="text-sm font-medium text-gray-700">Contact</label>
                      <div className="space-y-1">
                        <div className="flex items-center text-gray-900">
                          <Phone className="h-4 w-4 mr-2" />
                          {clinic.phone}
                        </div>
                        <p className="text-gray-900">{clinic.email}</p>
                      </div>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-gray-700">Rating</label>
                      <div className="flex items-center">
                        <Star className="h-4 w-4 text-yellow-500 fill-current mr-1" />
                        <span className="text-gray-900">{clinic.rating}/5</span>
                      </div>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-gray-700">Languages</label>
                      <div className="flex flex-wrap gap-1 mt-1">
                        {clinic.languages.map((lang) => (
                          <Badge key={lang} variant="secondary">
                            {lang}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  </div>
                </div>

                <div className="mt-6">
                  <label className="text-sm font-medium text-gray-700">Description</label>
                  <p className="text-gray-900 mt-1">{clinic.description}</p>
                </div>

                <div className="mt-6 flex gap-4">
                  <Button>Edit Clinic Info</Button>
                  <Button variant="outline">Manage Settings</Button>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}