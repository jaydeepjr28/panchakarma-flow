import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Calendar, Clock, MapPin, Star, LogOut, Phone, Mail, AlertCircle } from 'lucide-react';
import { User, Clinic, Doctor, TherapyPackage, Appointment } from '@/lib/types';
import { getStoredData } from '@/lib/data';
import BookingSystem from '@/components/BookingSystem';

interface PatientDashboardProps {
  user: User;
  onLogout: () => void;
}

export default function PatientDashboard({ user, onLogout }: PatientDashboardProps) {
  const [clinics, setClinics] = useState<Clinic[]>([]);
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [selectedClinic, setSelectedClinic] = useState<Clinic | null>(null);

  useEffect(() => {
    setClinics(getStoredData('clinics', []));
    const allAppointments = getStoredData('appointments', []);
    setAppointments(allAppointments.filter(apt => apt.patientId === user.id));
  }, [user.id]);

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

  const getStatusText = (status: string) => {
    switch (status) {
      case 'pending': return 'PENDING APPROVAL';
      case 'approved': return 'APPROVED';
      case 'scheduled': return 'SCHEDULED';
      case 'confirmed': return 'CONFIRMED';
      case 'checked_in': return 'CHECKED IN';
      case 'in_progress': return 'IN PROGRESS';
      case 'completed': return 'COMPLETED';
      case 'cancelled': return 'CANCELLED';
      case 'rejected': return 'REJECTED';
      default: return status.toUpperCase();
    }
  };

  const getStatusDescription = (status: string) => {
    switch (status) {
      case 'pending': return 'Your appointment request is being reviewed by the clinic';
      case 'approved': return 'Your appointment has been approved and will be scheduled soon';
      case 'scheduled': return 'Your appointment is scheduled and confirmed';
      case 'confirmed': return 'Your appointment is confirmed';
      case 'rejected': return 'Your appointment request was not approved';
      default: return '';
    }
  };

  if (selectedClinic) {
    return (
      <BookingSystem
        clinic={selectedClinic}
        patient={user}
        onBack={() => setSelectedClinic(null)}
        onBookingComplete={() => {
          setSelectedClinic(null);
          // Refresh appointments
          const allAppointments = getStoredData('appointments', []);
          setAppointments(allAppointments.filter(apt => apt.patientId === user.id));
        }}
      />
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Welcome, {user.name}</h1>
              <p className="text-gray-600">Manage your Panchakarma therapy journey</p>
            </div>
            <Button variant="outline" onClick={onLogout}>
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-8">
        <Tabs defaultValue="browse" className="space-y-6">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="browse">Browse Clinics</TabsTrigger>
            <TabsTrigger value="appointments">My Appointments</TabsTrigger>
            <TabsTrigger value="profile">Profile</TabsTrigger>
          </TabsList>

          <TabsContent value="browse" className="space-y-6">
            <div>
              <h2 className="text-xl font-semibold mb-4">Available Clinics</h2>
              <div className="grid md:grid-cols-2 gap-6">
                {clinics.map((clinic) => (
                  <Card key={clinic.id} className="hover:shadow-lg transition-shadow cursor-pointer">
                    <CardHeader>
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <CardTitle className="flex items-center gap-2">
                            {clinic.name}
                            <div className="flex items-center">
                              <Star className="h-4 w-4 text-yellow-500 fill-current" />
                              <span className="text-sm text-gray-600 ml-1">{clinic.rating}</span>
                            </div>
                          </CardTitle>
                          <CardDescription className="mt-2">{clinic.description}</CardDescription>
                        </div>
                        <img 
                          src={clinic.image} 
                          alt={clinic.name}
                          className="w-16 h-16 rounded-lg object-cover ml-4"
                        />
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2 text-sm text-gray-600">
                        <div className="flex items-center">
                          <MapPin className="h-4 w-4 mr-2" />
                          {clinic.address}
                        </div>
                        <div className="flex items-center">
                          <Clock className="h-4 w-4 mr-2" />
                          {clinic.hours}
                        </div>
                        <div className="flex items-center">
                          <Phone className="h-4 w-4 mr-2" />
                          {clinic.phone}
                        </div>
                      </div>
                      
                      <div className="mt-4">
                        <p className="text-sm font-medium mb-2">Languages:</p>
                        <div className="flex flex-wrap gap-1">
                          {clinic.languages.map((lang) => (
                            <Badge key={lang} variant="secondary" className="text-xs">
                              {lang}
                            </Badge>
                          ))}
                        </div>
                      </div>

                      <Button 
                        className="w-full mt-4"
                        onClick={() => setSelectedClinic(clinic)}
                      >
                        Book Appointment
                      </Button>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          </TabsContent>

          <TabsContent value="appointments" className="space-y-6">
            <div>
              <h2 className="text-xl font-semibold mb-4">My Appointments</h2>
              {appointments.length === 0 ? (
                <Card>
                  <CardContent className="text-center py-8">
                    <Calendar className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                    <p className="text-gray-600">No appointments scheduled yet.</p>
                    <p className="text-sm text-gray-500 mt-2">Browse clinics to book your first appointment.</p>
                  </CardContent>
                </Card>
              ) : (
                <div className="space-y-4">
                  {appointments
                    .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
                    .map((appointment) => {
                      const clinic = clinics.find(c => c.id === appointment.clinicId);
                      const doctors = getStoredData('doctors', []);
                      const doctor = doctors.find(d => d.id === appointment.doctorId);
                      const packages = getStoredData('packages', []);
                      const therapyPackage = packages.find(p => p.id === appointment.packageId);

                      return (
                        <Card key={appointment.id}>
                          <CardHeader>
                            <div className="flex items-start justify-between">
                              <div>
                                <CardTitle className="text-lg">{clinic?.name}</CardTitle>
                                <CardDescription>
                                  {therapyPackage?.name} with {doctor?.name}
                                </CardDescription>
                              </div>
                              <div className="text-right">
                                <Badge className={getStatusColor(appointment.status)}>
                                  {getStatusText(appointment.status)}
                                </Badge>
                                {appointment.status === 'pending' && (
                                  <div className="flex items-center mt-1 text-xs text-yellow-600">
                                    <AlertCircle className="h-3 w-3 mr-1" />
                                    Awaiting approval
                                  </div>
                                )}
                              </div>
                            </div>
                          </CardHeader>
                          <CardContent>
                            <div className="grid md:grid-cols-2 gap-4">
                              <div className="space-y-2">
                                <div className="flex items-center text-sm">
                                  <Calendar className="h-4 w-4 mr-2 text-gray-500" />
                                  {new Date(appointment.date).toLocaleDateString()}
                                </div>
                                <div className="flex items-center text-sm">
                                  <Clock className="h-4 w-4 mr-2 text-gray-500" />
                                  {appointment.time}
                                </div>
                              </div>
                              <div className="space-y-2">
                                <p className="text-sm text-gray-600">
                                  <strong>Duration:</strong> {therapyPackage?.durationMinutes} minutes
                                </p>
                                <p className="text-sm text-gray-600">
                                  <strong>Price:</strong> ₹{therapyPackage?.price}
                                </p>
                              </div>
                            </div>
                            
                            {/* Status-specific information */}
                            {appointment.status === 'pending' && (
                              <div className="mt-4 p-3 bg-yellow-50 rounded-lg">
                                <p className="text-sm text-yellow-800">
                                  <strong>Status:</strong> {getStatusDescription(appointment.status)}
                                </p>
                              </div>
                            )}

                            {appointment.status === 'approved' && (
                              <div className="mt-4 p-3 bg-green-50 rounded-lg">
                                <p className="text-sm text-green-800">
                                  <strong>Good news!</strong> Your appointment has been approved. The clinic will contact you soon with final confirmation details.
                                </p>
                              </div>
                            )}

                            {appointment.status === 'rejected' && appointment.rejectionReason && (
                              <div className="mt-4 p-3 bg-red-50 rounded-lg">
                                <p className="text-sm text-red-800">
                                  <strong>Reason:</strong> {appointment.rejectionReason}
                                </p>
                              </div>
                            )}

                            {(appointment.status === 'scheduled' || appointment.status === 'confirmed') && (
                              <div className="mt-4 p-3 bg-blue-50 rounded-lg">
                                <p className="text-sm text-blue-800">
                                  <strong>Pre-procedure checklist:</strong> Please follow the fasting and preparation guidelines sent to your email.
                                </p>
                              </div>
                            )}
                            
                            {appointment.notes && (
                              <div className="mt-4 p-3 bg-gray-50 rounded-lg">
                                <p className="text-sm text-gray-700">
                                  <strong>Notes:</strong> {appointment.notes}
                                </p>
                              </div>
                            )}

                            {/* Action buttons based on status */}
                            {appointment.status === 'pending' && (
                              <div className="mt-4 text-center">
                                <p className="text-xs text-gray-500">
                                  Submitted on {new Date(appointment.createdAt).toLocaleDateString()}
                                </p>
                              </div>
                            )}
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
                <CardTitle>Profile Information</CardTitle>
                <CardDescription>Your account details and preferences</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid md:grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium text-gray-700">Name</label>
                    <p className="text-gray-900">{user.name}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700">Email</label>
                    <p className="text-gray-900">{user.email}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700">Phone</label>
                    <p className="text-gray-900">{user.phone}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-gray-700">Role</label>
                    <p className="text-gray-900 capitalize">{user.role.replace('_', ' ')}</p>
                  </div>
                </div>
                
                <div className="mt-6 p-4 bg-green-50 rounded-lg">
                  <h4 className="font-medium text-green-800 mb-2">Health & Safety</h4>
                  <p className="text-sm text-green-700">
                    Always consult with qualified practitioners and inform them of any health conditions, 
                    allergies, or medications before starting Panchakarma therapy.
                  </p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}