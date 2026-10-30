import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Calendar, Clock, MapPin, Star, LogOut, Phone, Mail, AlertCircle, Sparkles } from 'lucide-react';
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
    // Sort appointments by createdAt in descending order (newest first)
    const userAppointments = allAppointments
      .filter(apt => apt.patientId === user.id)
      .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
    setAppointments(userAppointments);
  }, [user.id]);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'pending': return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'approved': return 'bg-green-100 text-green-800 border-green-200';
      case 'scheduled': return 'bg-blue-100 text-blue-800 border-blue-200';
      case 'confirmed': return 'bg-emerald-100 text-emerald-800 border-emerald-200';
      case 'checked_in': return 'bg-purple-100 text-purple-800 border-purple-200';
      case 'in_progress': return 'bg-indigo-100 text-indigo-800 border-indigo-200';
      case 'completed': return 'bg-gray-100 text-gray-800 border-gray-200';
      case 'cancelled': return 'bg-red-100 text-red-800 border-red-200';
      case 'rejected': return 'bg-red-100 text-red-800 border-red-200';
      default: return 'bg-gray-100 text-gray-800 border-gray-200';
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
          // Refresh appointments with proper sorting
          const allAppointments = getStoredData('appointments', []);
          const userAppointments = allAppointments
            .filter(apt => apt.patientId === user.id)
            .sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
          setAppointments(userAppointments);
        }}
      />
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-teal-50 to-cyan-50">
      {/* Header */}
      <div className="bg-white/80 backdrop-blur-sm shadow-sm border-b border-emerald-100">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-emerald-600 to-teal-600 bg-clip-text text-transparent">
                Welcome, {user.name}
              </h1>
              <p className="text-gray-600 mt-1">Manage your Panchakarma therapy journey</p>
            </div>
            <Button variant="outline" onClick={onLogout} className="hover:bg-emerald-50 border-emerald-200">
              <LogOut className="h-4 w-4 mr-2" />
              Logout
            </Button>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-12">
        <Tabs defaultValue="browse" className="space-y-8">
          <TabsList className="grid w-full grid-cols-3 bg-white/80 backdrop-blur-sm border border-emerald-100 rounded-2xl p-1">
            <TabsTrigger value="browse" className="rounded-xl data-[state=active]:bg-gradient-to-r data-[state=active]:from-emerald-500 data-[state=active]:to-teal-600 data-[state=active]:text-white font-semibold">
              Browse Clinics
            </TabsTrigger>
            <TabsTrigger value="appointments" className="rounded-xl data-[state=active]:bg-gradient-to-r data-[state=active]:from-emerald-500 data-[state=active]:to-teal-600 data-[state=active]:text-white font-semibold">
              My Appointments
            </TabsTrigger>
            <TabsTrigger value="profile" className="rounded-xl data-[state=active]:bg-gradient-to-r data-[state=active]:from-emerald-500 data-[state=active]:to-teal-600 data-[state=active]:text-white font-semibold">
              Profile
            </TabsTrigger>
          </TabsList>

          <TabsContent value="browse" className="space-y-8 animate-in fade-in duration-500">
            <div>
              <h2 className="text-2xl font-bold text-gray-800 mb-6">Available Clinics</h2>
              <div className="grid md:grid-cols-2 gap-8">
                {clinics.map((clinic) => (
                  <Card key={clinic.id} className="hover:shadow-2xl transition-all duration-500 hover:-translate-y-2 cursor-pointer border-0 bg-white/80 backdrop-blur-sm">
                    <CardHeader>
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <CardTitle className="flex items-center gap-3 text-xl">
                            {clinic.name}
                            <div className="flex items-center bg-yellow-100 px-2 py-1 rounded-full">
                              <Star className="h-4 w-4 text-yellow-500 fill-current" />
                              <span className="text-sm text-yellow-700 ml-1 font-semibold">{clinic.rating}</span>
                            </div>
                          </CardTitle>
                          <CardDescription className="mt-3 text-gray-600 leading-relaxed">{clinic.description}</CardDescription>
                        </div>
                        <img 
                          src={clinic.image} 
                          alt={clinic.name}
                          className="w-20 h-20 rounded-2xl object-cover ml-4 shadow-md"
                        />
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3 text-sm text-gray-600 mb-6">
                        <div className="flex items-center">
                          <MapPin className="h-4 w-4 mr-3 text-emerald-500" />
                          {clinic.address}
                        </div>
                        <div className="flex items-center">
                          <Clock className="h-4 w-4 mr-3 text-emerald-500" />
                          {clinic.hours}
                        </div>
                        <div className="flex items-center">
                          <Phone className="h-4 w-4 mr-3 text-emerald-500" />
                          {clinic.phone}
                        </div>
                      </div>
                      
                      <div className="mb-6">
                        <p className="text-sm font-semibold mb-3 text-gray-700">Languages:</p>
                        <div className="flex flex-wrap gap-2">
                          {clinic.languages.map((lang) => (
                            <Badge key={lang} variant="secondary" className="text-xs bg-emerald-100 text-emerald-700 hover:bg-emerald-200">
                              {lang}
                            </Badge>
                          ))}
                        </div>
                      </div>

                      <Button 
                        className="w-full bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white font-semibold py-3 rounded-xl transition-all duration-300 shadow-lg hover:shadow-xl"
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

          <TabsContent value="appointments" className="space-y-8 animate-in fade-in duration-500">
            <div>
              <div className="flex items-center justify-between mb-6">
                <h2 className="text-2xl font-bold text-gray-800">My Appointments</h2>
                {appointments.length > 0 && (
                  <div className="flex items-center text-sm text-gray-600">
                    <Sparkles className="h-4 w-4 mr-2 text-emerald-500" />
                    Sorted by newest first
                  </div>
                )}
              </div>
              {appointments.length === 0 ? (
                <Card className="border-0 bg-white/80 backdrop-blur-sm shadow-lg">
                  <CardContent className="text-center py-16">
                    <Calendar className="h-16 w-16 text-gray-300 mx-auto mb-6" />
                    <h3 className="text-xl font-semibold text-gray-700 mb-2">No appointments scheduled yet</h3>
                    <p className="text-gray-500 mb-6">Browse clinics to book your first appointment and start your wellness journey.</p>
                    <Button 
                      onClick={() => (document.querySelector('[value="browse"]') as HTMLElement | null)?.click()}
                      className="bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white font-semibold px-6 py-3 rounded-xl"
                    >
                      Browse Clinics
                    </Button>
                  </CardContent>
                </Card>
              ) : (
                <div className="space-y-6">
                  {appointments.map((appointment, index) => {
                    const clinic = clinics.find(c => c.id === appointment.clinicId);
                    const doctors = getStoredData('doctors', []);
                    const doctor = doctors.find(d => d.id === appointment.doctorId);
                    const packages = getStoredData('packages', []);
                    const therapyPackage = packages.find(p => p.id === appointment.packageId);

                    return (
                      <Card key={appointment.id} className={`border-0 bg-white/90 backdrop-blur-sm shadow-lg hover:shadow-xl transition-all duration-300 animate-in slide-in-from-left duration-${300 + index * 100}`}>
                        <CardHeader>
                          <div className="flex items-start justify-between">
                            <div>
                              <CardTitle className="text-xl font-bold text-gray-800 flex items-center">
                                {clinic?.name}
                                {index === 0 && (
                                  <Badge className="ml-3 bg-gradient-to-r from-emerald-400 to-teal-500 text-white border-0">
                                    Latest
                                  </Badge>
                                )}
                              </CardTitle>
                              <CardDescription className="mt-2 text-gray-600">
                                {therapyPackage?.name} with {doctor?.name}
                              </CardDescription>
                            </div>
                            <div className="text-right">
                              <Badge className={`${getStatusColor(appointment.status)} border font-semibold`}>
                                {getStatusText(appointment.status)}
                              </Badge>
                              {appointment.status === 'pending' && (
                                <div className="flex items-center mt-2 text-xs text-yellow-600">
                                  <AlertCircle className="h-3 w-3 mr-1" />
                                  Awaiting approval
                                </div>
                              )}
                            </div>
                          </div>
                        </CardHeader>
                        <CardContent>
                          <div className="grid md:grid-cols-2 gap-6 mb-6">
                            <div className="space-y-3">
                              <div className="flex items-center text-sm bg-gray-50 p-3 rounded-xl">
                                <Calendar className="h-4 w-4 mr-3 text-emerald-500" />
                                <span className="font-medium">{new Date(appointment.date).toLocaleDateString()}</span>
                              </div>
                              <div className="flex items-center text-sm bg-gray-50 p-3 rounded-xl">
                                <Clock className="h-4 w-4 mr-3 text-emerald-500" />
                                <span className="font-medium">{appointment.time}</span>
                              </div>
                            </div>
                            <div className="space-y-3">
                              <div className="bg-gray-50 p-3 rounded-xl">
                                <p className="text-sm text-gray-600">
                                  <strong>Duration:</strong> {therapyPackage?.durationMinutes} minutes
                                </p>
                              </div>
                              <div className="bg-gradient-to-r from-emerald-50 to-teal-50 p-3 rounded-xl border border-emerald-100">
                                <p className="text-sm text-emerald-800">
                                  <strong>Price:</strong> ₹{therapyPackage?.price}
                                </p>
                              </div>
                            </div>
                          </div>
                          
                          {/* Status-specific information */}
                          {appointment.status === 'pending' && (
                            <div className="p-4 bg-gradient-to-r from-yellow-50 to-orange-50 rounded-2xl border border-yellow-200">
                              <p className="text-sm text-yellow-800">
                                <strong>Status:</strong> {getStatusDescription(appointment.status)}
                              </p>
                            </div>
                          )}

                          {appointment.status === 'approved' && (
                            <div className="p-4 bg-gradient-to-r from-green-50 to-emerald-50 rounded-2xl border border-green-200">
                              <p className="text-sm text-green-800">
                                <strong>Good news!</strong> Your appointment has been approved. The clinic will contact you soon with final confirmation details.
                              </p>
                            </div>
                          )}

                          {appointment.status === 'rejected' && appointment.rejectionReason && (
                            <div className="p-4 bg-gradient-to-r from-red-50 to-pink-50 rounded-2xl border border-red-200">
                              <p className="text-sm text-red-800">
                                <strong>Reason:</strong> {appointment.rejectionReason}
                              </p>
                            </div>
                          )}

                          {(appointment.status === 'scheduled' || appointment.status === 'confirmed') && (
                            <div className="p-4 bg-gradient-to-r from-blue-50 to-indigo-50 rounded-2xl border border-blue-200">
                              <p className="text-sm text-blue-800">
                                <strong>Pre-procedure checklist:</strong> Please follow the fasting and preparation guidelines sent to your email.
                              </p>
                            </div>
                          )}
                          
                          {appointment.notes && (
                            <div className="mt-4 p-4 bg-gray-50 rounded-2xl border border-gray-200">
                              <p className="text-sm text-gray-700">
                                <strong>Notes:</strong> {appointment.notes}
                              </p>
                            </div>
                          )}

                          {/* Action buttons based on status */}
                          {appointment.status === 'pending' && (
                            <div className="mt-6 text-center">
                              <p className="text-xs text-gray-500">
                                Submitted on {new Date(appointment.createdAt).toLocaleDateString()} at {new Date(appointment.createdAt).toLocaleTimeString()}
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

          <TabsContent value="profile" className="space-y-8 animate-in fade-in duration-500">
            <Card className="border-0 bg-white/90 backdrop-blur-sm shadow-lg">
              <CardHeader>
                <CardTitle className="text-2xl font-bold text-gray-800">Profile Information</CardTitle>
                <CardDescription className="text-gray-600">Your account details and preferences</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid md:grid-cols-2 gap-6">
                  <div className="bg-gray-50 p-4 rounded-xl">
                    <label className="text-sm font-semibold text-gray-700 block mb-1">Name</label>
                    <p className="text-gray-900 font-medium">{user.name}</p>
                  </div>
                  <div className="bg-gray-50 p-4 rounded-xl">
                    <label className="text-sm font-semibold text-gray-700 block mb-1">Email</label>
                    <p className="text-gray-900 font-medium">{user.email}</p>
                  </div>
                  <div className="bg-gray-50 p-4 rounded-xl">
                    <label className="text-sm font-semibold text-gray-700 block mb-1">Phone</label>
                    <p className="text-gray-900 font-medium">{user.phone}</p>
                  </div>
                  <div className="bg-gray-50 p-4 rounded-xl">
                    <label className="text-sm font-semibold text-gray-700 block mb-1">Role</label>
                    <p className="text-gray-900 font-medium capitalize">{user.role.replace('_', ' ')}</p>
                  </div>
                </div>
                
                <div className="p-6 bg-gradient-to-r from-green-50 to-emerald-50 rounded-2xl border border-green-200">
                  <h4 className="font-semibold text-green-800 mb-3 flex items-center">
                    <Sparkles className="h-5 w-5 mr-2" />
                    Health & Safety
                  </h4>
                  <p className="text-sm text-green-700 leading-relaxed">
                    Always consult with qualified practitioners and inform them of any health conditions, 
                    allergies, or medications before starting Panchakarma therapy. Your safety and well-being are our top priority.
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