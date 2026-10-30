import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar } from '@/components/ui/calendar';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { ArrowLeft, Clock, Star, User, Calendar as CalendarIcon, CheckCircle, Sparkles, Heart, Shield } from 'lucide-react';
import { Clinic, Doctor, TherapyPackage, User as UserType, Appointment } from '@/lib/types';
import { getStoredData, setStoredData, generateId, formatDate, formatTime } from '@/lib/data';

interface BookingSystemProps {
  clinic: Clinic;
  patient: UserType;
  onBack: () => void;
  onBookingComplete: () => void;
}

export default function BookingSystem({ clinic, patient, onBack, onBookingComplete }: BookingSystemProps) {
  const [doctors, setDoctors] = useState<Doctor[]>([]);
  const [packages, setPackages] = useState<TherapyPackage[]>([]);
  const [selectedPackage, setSelectedPackage] = useState<TherapyPackage | null>(null);
  const [selectedDoctor, setSelectedDoctor] = useState<Doctor | null>(null);
  const [selectedDate, setSelectedDate] = useState<Date | undefined>(undefined);
  const [selectedTime, setSelectedTime] = useState<string>('');
  const [notes, setNotes] = useState<string>('');
  const [availableSlots, setAvailableSlots] = useState<string[]>([]);
  const [step, setStep] = useState<'package' | 'doctor' | 'datetime' | 'confirm' | 'success'>('package');

  useEffect(() => {
    const allDoctors = getStoredData('doctors', []);
    const allPackages = getStoredData('packages', []);
    
    setDoctors(allDoctors.filter(d => d.clinicId === clinic.id));
    setPackages(allPackages.filter(p => p.clinicId === clinic.id));
  }, [clinic.id]);

  useEffect(() => {
    if (selectedDate) {
      generateAvailableSlots();
    }
  }, [selectedDoctor, selectedDate]);

  const generateAvailableSlots = () => {
    if (!selectedDate) return;

    // Get day name in lowercase format
    const dayNames = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];
    const dayName = dayNames[selectedDate.getDay()];
    
    // If no doctor is selected, use the first available doctor or all doctors
    const doctorsToCheck = selectedDoctor ? [selectedDoctor] : doctors;
    
    if (doctorsToCheck.length === 0) {
      setAvailableSlots([]);
      return;
    }

    const allSlots: string[] = [];
    const existingAppointments = getStoredData('appointments', []);
    const dateStr = formatDate(selectedDate);

    doctorsToCheck.forEach(doctor => {
      const schedule = doctor.schedule[dayName];
      
      if (schedule && schedule.length > 0) {
        // Get existing appointments for this doctor on this date
        const doctorAppointments = existingAppointments.filter(
          apt => apt.doctorId === doctor.id && 
                 apt.date === dateStr && 
                 apt.status !== 'cancelled' &&
                 apt.status !== 'rejected'
        );

        schedule.forEach(timeSlot => {
          const startHour = parseInt(timeSlot.start.split(':')[0]);
          const startMinute = parseInt(timeSlot.start.split(':')[1]);
          const endHour = parseInt(timeSlot.end.split(':')[0]);
          
          // Generate 1-hour slots
          for (let hour = startHour; hour < endHour; hour++) {
            const slotTime = `${hour.toString().padStart(2, '0')}:${startMinute.toString().padStart(2, '0')}`;
            
            // Check if slot is already booked for this doctor
            const isBooked = doctorAppointments.some(apt => apt.time === slotTime);
            
            if (!isBooked && !allSlots.includes(slotTime)) {
              allSlots.push(slotTime);
            }
          }
        });
      }
    });

    // Sort slots chronologically
    allSlots.sort();
    setAvailableSlots(allSlots);
  };

  const handleBooking = () => {
    if (!selectedPackage || !selectedDate || !selectedTime) return;

    // Use selected doctor or auto-assign first available doctor
    const doctorToUse = selectedDoctor || doctors[0];
    if (!doctorToUse) return;

    // Create a pending appointment request
    const appointment: Appointment = {
      id: generateId(),
      patientId: patient.id,
      clinicId: clinic.id,
      doctorId: doctorToUse.id,
      packageId: selectedPackage.id,
      date: formatDate(selectedDate),
      time: selectedTime,
      status: 'pending',
      notes: notes.trim() || undefined,
      createdAt: new Date().toISOString()
    };

    const appointments = getStoredData('appointments', []);
    appointments.push(appointment);
    setStoredData('appointments', appointments);

    // Create notification for patient
    const notifications = getStoredData('notifications', []);
    notifications.push({
      id: generateId(),
      userId: patient.id,
      title: 'Booking Request Submitted',
      message: `Your appointment request at ${clinic.name} has been submitted for ${selectedDate.toLocaleDateString()} at ${selectedTime}. You will be notified once the clinic approves your request.`,
      type: 'info',
      read: false,
      createdAt: new Date().toISOString()
    });

    // Create notification for clinic admin
    const clinicUsers = getStoredData('users', []);
    const clinicAdmin = clinicUsers.find(u => u.clinicId === clinic.id && u.role === 'clinic_admin');
    
    if (clinicAdmin) {
      notifications.push({
        id: generateId(),
        userId: clinicAdmin.id,
        title: 'New Appointment Request',
        message: `New appointment request from ${patient.name} for ${selectedPackage.name} on ${selectedDate.toLocaleDateString()} at ${selectedTime}`,
        type: 'info',
        read: false,
        createdAt: new Date().toISOString()
      });
    }

    setStoredData('notifications', notifications);
    setStep('success');
  };

  const isDateDisabled = (date: Date) => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    // Disable past dates
    if (date < today) return true;
    
    // Disable dates more than 30 days in future
    const maxDate = new Date();
    maxDate.setDate(maxDate.getDate() + 30);
    if (date > maxDate) return true;

    // Check if any doctor has schedule for this day
    const dayNames = ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'];
    const dayName = dayNames[date.getDay()];
    
    // If no doctor selected, check if any doctor is available
    const doctorsToCheck = selectedDoctor ? [selectedDoctor] : doctors;
    
    return !doctorsToCheck.some(doctor => {
      const schedule = doctor.schedule[dayName];
      return schedule && schedule.length > 0;
    });
  };

  if (step === 'success') {
    return (
      <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-teal-50 to-cyan-50">
        {/* Header */}
        <div className="bg-white/80 backdrop-blur-sm shadow-sm border-b border-emerald-100">
          <div className="container mx-auto px-4 py-6">
            <div className="flex items-center">
              <Button variant="ghost" onClick={onBack} className="mr-4 hover:bg-emerald-50 transition-colors">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Clinics
              </Button>
              <div>
                <h1 className="text-3xl font-bold bg-gradient-to-r from-emerald-600 to-teal-600 bg-clip-text text-transparent">
                  Booking Request Submitted
                </h1>
                <p className="text-gray-600 mt-1">{clinic.name}</p>
              </div>
            </div>
          </div>
        </div>

        <div className="container mx-auto px-4 py-12">
          <div className="max-w-2xl mx-auto">
            <Card className="border-0 shadow-2xl bg-white/90 backdrop-blur-sm">
              <CardHeader className="text-center pb-8">
                <div className="relative w-20 h-20 bg-gradient-to-br from-emerald-400 to-teal-500 rounded-full flex items-center justify-center mx-auto mb-6 shadow-lg">
                  <CheckCircle className="h-10 w-10 text-white" />
                  <div className="absolute -top-1 -right-1 w-6 h-6 bg-yellow-400 rounded-full flex items-center justify-center">
                    <Sparkles className="h-3 w-3 text-yellow-800" />
                  </div>
                </div>
                <CardTitle className="text-3xl font-bold text-gray-800 mb-2">
                  Request Submitted Successfully!
                </CardTitle>
                <CardDescription className="text-lg text-gray-600">
                  Your appointment request has been sent to {clinic.name} for approval.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-8">
                <div className="bg-gradient-to-r from-blue-50 to-indigo-50 p-6 rounded-2xl border border-blue-100">
                  <h4 className="font-semibold text-blue-900 mb-3 flex items-center">
                    <Heart className="h-5 w-5 mr-2 text-blue-600" />
                    What happens next?
                  </h4>
                  <ul className="text-sm text-blue-800 space-y-2">
                    <li className="flex items-start">
                      <div className="w-2 h-2 bg-blue-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                      The clinic will review your appointment request
                    </li>
                    <li className="flex items-start">
                      <div className="w-2 h-2 bg-blue-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                      You'll receive a notification once it's approved or if any changes are needed
                    </li>
                    <li className="flex items-start">
                      <div className="w-2 h-2 bg-blue-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                      Once approved, your appointment will be confirmed and assigned to a therapist
                    </li>
                    <li className="flex items-start">
                      <div className="w-2 h-2 bg-blue-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                      You can track the status in your "My Appointments" section
                    </li>
                  </ul>
                </div>

                <div className="border border-gray-200 rounded-2xl p-6 bg-gradient-to-br from-gray-50 to-white">
                  <h4 className="font-semibold text-gray-900 mb-4 flex items-center">
                    <Shield className="h-5 w-5 mr-2 text-gray-600" />
                    Request Summary
                  </h4>
                  <div className="grid md:grid-cols-2 gap-4 text-sm">
                    <div className="space-y-1">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Package</span>
                      <p className="font-semibold text-gray-900">{selectedPackage?.name}</p>
                    </div>
                    <div className="space-y-1">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Doctor</span>
                      <p className="font-semibold text-gray-900">
                        {selectedDoctor ? selectedDoctor.name : 'Auto-assigned'}
                      </p>
                    </div>
                    <div className="space-y-1">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Date & Time</span>
                      <p className="font-semibold text-gray-900">
                        {selectedDate?.toLocaleDateString()} at {selectedTime}
                      </p>
                    </div>
                    <div className="space-y-1">
                      <span className="text-gray-500 text-xs uppercase tracking-wide">Price</span>
                      <p className="font-bold text-lg text-emerald-600">₹{selectedPackage?.price}</p>
                    </div>
                  </div>
                </div>

                <div className="flex gap-4">
                  <Button 
                    onClick={onBookingComplete} 
                    className="flex-1 bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white py-3 rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl"
                  >
                    Go to My Appointments
                  </Button>
                  <Button 
                    variant="outline" 
                    onClick={onBack} 
                    className="flex-1 border-2 border-gray-200 hover:border-emerald-300 hover:bg-emerald-50 py-3 rounded-xl font-semibold transition-all duration-300"
                  >
                    Browse More Clinics
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-teal-50 to-cyan-50">
      {/* Header */}
      <div className="bg-white/80 backdrop-blur-sm shadow-sm border-b border-emerald-100">
        <div className="container mx-auto px-4 py-6">
          <div className="flex items-center">
            <Button variant="ghost" onClick={onBack} className="mr-4 hover:bg-emerald-50 transition-colors">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-emerald-600 to-teal-600 bg-clip-text text-transparent">
                Book Appointment
              </h1>
              <p className="text-gray-600 mt-1">{clinic.name}</p>
            </div>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-12">
        {/* Progress Steps */}
        <div className="flex items-center justify-center mb-12">
          <div className="flex items-center space-x-6">
            {[
              { name: 'package', label: 'Package' },
              { name: 'doctor', label: 'Doctor' },
              { name: 'datetime', label: 'Date & Time' },
              { name: 'confirm', label: 'Confirm' }
            ].map((stepInfo, index) => (
              <div key={stepInfo.name} className="flex items-center">
                <div className="flex flex-col items-center">
                  <div className={`w-12 h-12 rounded-full flex items-center justify-center text-sm font-bold transition-all duration-300 ${
                    step === stepInfo.name 
                      ? 'bg-gradient-to-br from-emerald-500 to-teal-600 text-white shadow-lg scale-110' 
                      : ['package', 'doctor', 'datetime', 'confirm'].indexOf(step) > index 
                        ? 'bg-gradient-to-br from-green-500 to-emerald-600 text-white shadow-md' 
                        : 'bg-white text-gray-400 border-2 border-gray-200 shadow-sm'
                  }`}>
                    {index + 1}
                  </div>
                  <span className={`text-xs mt-2 font-medium ${
                    step === stepInfo.name ? 'text-emerald-600' : 'text-gray-500'
                  }`}>
                    {stepInfo.label}
                  </span>
                </div>
                {index < 3 && (
                  <div className={`w-16 h-1 mx-4 rounded-full transition-colors duration-300 ${
                    ['package', 'doctor', 'datetime', 'confirm'].indexOf(step) > index 
                      ? 'bg-gradient-to-r from-emerald-400 to-teal-500' 
                      : 'bg-gray-200'
                  }`} />
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Step Content */}
        <div className="max-w-6xl mx-auto">
          {step === 'package' && (
            <div className="animate-in fade-in duration-500">
              <h2 className="text-2xl font-bold text-gray-800 mb-8 text-center">
                Select Your Therapy Package
              </h2>
              <div className="grid md:grid-cols-2 gap-8">
                {packages.map((pkg) => (
                  <Card 
                    key={pkg.id} 
                    className={`cursor-pointer transition-all duration-300 border-2 hover:shadow-2xl hover:-translate-y-1 ${
                      selectedPackage?.id === pkg.id 
                        ? 'ring-4 ring-emerald-200 bg-gradient-to-br from-emerald-50 to-teal-50 border-emerald-300 shadow-xl' 
                        : 'hover:border-emerald-200 bg-white/80 backdrop-blur-sm border-gray-200'
                    }`}
                    onClick={() => setSelectedPackage(pkg)}
                  >
                    <CardHeader className="pb-4">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <CardTitle className="text-xl font-bold text-gray-800 mb-2">{pkg.name}</CardTitle>
                          <CardDescription className="text-gray-600 leading-relaxed">{pkg.description}</CardDescription>
                        </div>
                        <img 
                          src={pkg.image} 
                          alt={pkg.name}
                          className="w-20 h-20 rounded-2xl object-cover ml-4 shadow-md"
                        />
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-4">
                        <div className="grid grid-cols-3 gap-4 text-center">
                          <div className="bg-white/60 rounded-xl p-3">
                            <span className="text-xs text-gray-500 block">Duration</span>
                            <span className="font-bold text-gray-800">{pkg.durationMinutes}min</span>
                          </div>
                          <div className="bg-white/60 rounded-xl p-3">
                            <span className="text-xs text-gray-500 block">Sessions</span>
                            <span className="font-bold text-gray-800">{pkg.totalSessions}</span>
                          </div>
                          <div className="bg-gradient-to-br from-emerald-100 to-teal-100 rounded-xl p-3">
                            <span className="text-xs text-emerald-700 block">Price</span>
                            <span className="font-bold text-lg text-emerald-600">₹{pkg.price}</span>
                          </div>
                        </div>
                        
                        <div className="bg-white/40 rounded-xl p-4">
                          <p className="text-sm font-semibold mb-3 text-gray-700">Treatment Steps:</p>
                          <div className="space-y-2">
                            {pkg.steps.slice(0, 3).map((step, index) => (
                              <div key={step.id} className="flex items-center text-xs text-gray-600">
                                <div className="w-6 h-6 bg-emerald-100 text-emerald-600 rounded-full flex items-center justify-center mr-2 text-xs font-bold">
                                  {index + 1}
                                </div>
                                {step.name}
                              </div>
                            ))}
                            {pkg.steps.length > 3 && (
                              <div className="text-xs text-gray-500 ml-8">
                                +{pkg.steps.length - 3} more steps
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
              
              {selectedPackage && (
                <div className="mt-8 text-center animate-in slide-in-from-bottom duration-300">
                  <Button 
                    onClick={() => setStep('doctor')}
                    className="bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white px-8 py-3 rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl"
                  >
                    Continue to Doctor Selection
                  </Button>
                </div>
              )}
            </div>
          )}

          {step === 'doctor' && (
            <div className="animate-in fade-in duration-500">
              <h2 className="text-2xl font-bold text-gray-800 mb-8 text-center">
                Choose Your Doctor
              </h2>
              <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
                <Card 
                  className={`cursor-pointer transition-all duration-300 border-2 hover:shadow-2xl hover:-translate-y-1 ${
                    selectedDoctor === null 
                      ? 'ring-4 ring-emerald-200 bg-gradient-to-br from-emerald-50 to-teal-50 border-emerald-300 shadow-xl' 
                      : 'hover:border-emerald-200 bg-white/80 backdrop-blur-sm border-gray-200'
                  }`}
                  onClick={() => setSelectedDoctor(null)}
                >
                  <CardHeader>
                    <CardTitle className="flex items-center text-lg">
                      <div className="w-10 h-10 bg-gradient-to-br from-blue-400 to-purple-500 rounded-full flex items-center justify-center mr-3">
                        <Sparkles className="h-5 w-5 text-white" />
                      </div>
                      Auto-Assign Doctor
                    </CardTitle>
                    <CardDescription className="leading-relaxed">
                      Let us assign the best available doctor for your appointment
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-gray-600 leading-relaxed">
                      Our intelligent system will automatically assign a qualified doctor based on availability and expertise.
                    </p>
                  </CardContent>
                </Card>

                {doctors.map((doctor) => (
                  <Card 
                    key={doctor.id} 
                    className={`cursor-pointer transition-all duration-300 border-2 hover:shadow-2xl hover:-translate-y-1 ${
                      selectedDoctor?.id === doctor.id 
                        ? 'ring-4 ring-emerald-200 bg-gradient-to-br from-emerald-50 to-teal-50 border-emerald-300 shadow-xl' 
                        : 'hover:border-emerald-200 bg-white/80 backdrop-blur-sm border-gray-200'
                    }`}
                    onClick={() => setSelectedDoctor(doctor)}
                  >
                    <CardHeader>
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <CardTitle className="flex items-center text-lg mb-2">
                            {doctor.name}
                            <div className="flex items-center ml-2 bg-yellow-100 px-2 py-1 rounded-full">
                              <Star className="h-3 w-3 text-yellow-500 fill-current" />
                              <span className="text-xs text-yellow-700 ml-1 font-semibold">{doctor.rating}</span>
                            </div>
                          </CardTitle>
                          <CardDescription className="text-sm">{doctor.qualifications}</CardDescription>
                        </div>
                        <img 
                          src={doctor.image} 
                          alt={doctor.name}
                          className="w-14 h-14 rounded-full object-cover ml-4 border-3 border-white shadow-md"
                        />
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-3">
                        <div className="bg-white/60 rounded-lg p-2">
                          <span className="text-xs text-gray-500">Experience: </span>
                          <span className="font-semibold text-gray-700">{doctor.experience} years</span>
                        </div>
                        <div>
                          <span className="text-xs text-gray-500 block mb-2">Specialties:</span>
                          <div className="flex flex-wrap gap-1">
                            {doctor.specialties.map((specialty) => (
                              <Badge key={specialty} variant="secondary" className="text-xs bg-emerald-100 text-emerald-700 hover:bg-emerald-200">
                                {specialty}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
              
              <div className="mt-8 text-center">
                <Button 
                  onClick={() => setStep('datetime')}
                  className="bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white px-8 py-3 rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl"
                >
                  Continue to Date & Time
                </Button>
              </div>
            </div>
          )}

          {step === 'datetime' && (
            <div className="animate-in fade-in duration-500">
              <h2 className="text-2xl font-bold text-gray-800 mb-8 text-center">
                Select Date & Time
              </h2>
              <div className="grid lg:grid-cols-2 gap-8">
                <Card className="border-2 border-gray-200 bg-white/80 backdrop-blur-sm shadow-lg">
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <CalendarIcon className="h-6 w-6 mr-3 text-emerald-600" />
                      Select Date
                    </CardTitle>
                    <CardDescription className="text-gray-600">
                      Choose your preferred appointment date
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="flex justify-center">
                    <Calendar
                      mode="single"
                      selected={selectedDate}
                      onSelect={setSelectedDate}
                      disabled={isDateDisabled}
                      className="rounded-2xl border-2 border-emerald-100 w-full max-w-sm bg-white shadow-sm"
                      classNames={{
                        months: "flex flex-col sm:flex-row space-y-4 sm:space-x-4 sm:space-y-0",
                        month: "space-y-4",
                        caption: "flex justify-center pt-1 relative items-center",
                        caption_label: "text-sm font-medium",
                        nav: "space-x-1 flex items-center",
                        nav_button: "h-7 w-7 bg-transparent p-0 opacity-50 hover:opacity-100",
                        nav_button_previous: "absolute left-1",
                        nav_button_next: "absolute right-1",
                        table: "w-full border-collapse space-y-1",
                        head_row: "flex",
                        head_cell: "text-muted-foreground rounded-md w-9 font-normal text-[0.8rem]",
                        row: "flex w-full mt-2",
                        cell: "text-center text-sm p-0 relative [&:has([aria-selected])]:bg-accent first:[&:has([aria-selected])]:rounded-l-md last:[&:has([aria-selected])]:rounded-r-md focus-within:relative focus-within:z-20",
                        day: "h-9 w-9 p-0 font-normal aria-selected:opacity-100 hover:bg-emerald-50 rounded-lg transition-colors",
                        day_selected: "bg-gradient-to-br from-emerald-500 to-teal-600 text-white hover:bg-emerald-600 hover:text-white focus:bg-emerald-600 focus:text-white shadow-md",
                        day_today: "bg-emerald-100 text-emerald-700 font-semibold",
                        day_outside: "text-muted-foreground opacity-50",
                        day_disabled: "text-muted-foreground opacity-30",
                        day_range_middle: "aria-selected:bg-accent aria-selected:text-accent-foreground",
                        day_hidden: "invisible",
                      }}
                    />
                  </CardContent>
                </Card>

                <Card className="border-2 border-gray-200 bg-white/80 backdrop-blur-sm shadow-lg">
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <Clock className="h-6 w-6 mr-3 text-emerald-600" />
                      Select Time
                    </CardTitle>
                    <CardDescription className="text-gray-600">
                      {selectedDate ? 
                        `Available slots for ${selectedDate.toLocaleDateString()}` : 
                        'Please select a date first'
                      }
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    {selectedDate ? (
                      availableSlots.length > 0 ? (
                        <div className="grid grid-cols-2 gap-3">
                          {availableSlots.map((slot) => (
                            <Button
                              key={slot}
                              variant={selectedTime === slot ? "default" : "outline"}
                              className={`text-sm h-12 rounded-xl font-semibold transition-all duration-300 ${
                                selectedTime === slot 
                                  ? 'bg-gradient-to-br from-emerald-500 to-teal-600 text-white shadow-lg hover:shadow-xl' 
                                  : 'border-2 border-gray-200 hover:border-emerald-300 hover:bg-emerald-50'
                              }`}
                              onClick={() => setSelectedTime(slot)}
                            >
                              {slot}
                            </Button>
                          ))}
                        </div>
                      ) : (
                        <div className="text-center py-12">
                          <Clock className="h-16 w-16 text-gray-300 mx-auto mb-4" />
                          <p className="text-gray-600 mb-2 font-semibold">No available slots for this date.</p>
                          <p className="text-sm text-gray-500">Please select another date or try auto-assigning a doctor.</p>
                        </div>
                      )
                    ) : (
                      <div className="text-center py-12">
                        <CalendarIcon className="h-16 w-16 text-gray-300 mx-auto mb-4" />
                        <p className="text-gray-600 font-semibold">Please select a date first.</p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </div>

              <Card className="mt-8 border-2 border-gray-200 bg-white/80 backdrop-blur-sm shadow-lg">
                <CardHeader>
                  <CardTitle className="text-xl">Additional Notes (Optional)</CardTitle>
                  <CardDescription className="text-gray-600">
                    Any specific requirements, health conditions, or preferences
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <Textarea
                    placeholder="e.g., I have back pain, prefer morning sessions, allergic to sesame oil..."
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    rows={4}
                    className="border-2 border-gray-200 rounded-xl focus:border-emerald-300 focus:ring-emerald-200 resize-none"
                  />
                </CardContent>
              </Card>
              
              {selectedDate && selectedTime && (
                <div className="mt-8 text-center animate-in slide-in-from-bottom duration-300">
                  <Button 
                    onClick={() => setStep('confirm')} 
                    size="lg"
                    className="bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white px-8 py-4 rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl"
                  >
                    Review Booking Request
                  </Button>
                </div>
              )}
            </div>
          )}

          {step === 'confirm' && selectedPackage && selectedDate && selectedTime && (
            <div className="animate-in fade-in duration-500">
              <h2 className="text-2xl font-bold text-gray-800 mb-8 text-center">
                Confirm Your Booking Request
              </h2>
              <Card className="border-2 border-gray-200 bg-white/90 backdrop-blur-sm shadow-xl max-w-4xl mx-auto">
                <CardHeader>
                  <CardTitle className="text-2xl text-center">Request Summary</CardTitle>
                  <CardDescription className="text-center text-gray-600">
                    This will be sent to the clinic for approval. You'll be notified once it's reviewed.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-6">
                  <div className="grid md:grid-cols-2 gap-6">
                    <div className="bg-gradient-to-br from-gray-50 to-white rounded-xl p-4 border border-gray-200">
                      <h4 className="font-semibold text-gray-900 mb-1">Clinic</h4>
                      <p className="text-gray-700">{clinic.name}</p>
                    </div>
                    <div className="bg-gradient-to-br from-gray-50 to-white rounded-xl p-4 border border-gray-200">
                      <h4 className="font-semibold text-gray-900 mb-1">Package</h4>
                      <p className="text-gray-700">{selectedPackage.name}</p>
                    </div>
                    <div className="bg-gradient-to-br from-gray-50 to-white rounded-xl p-4 border border-gray-200">
                      <h4 className="font-semibold text-gray-900 mb-1">Doctor</h4>
                      <p className="text-gray-700">
                        {selectedDoctor ? selectedDoctor.name : 'Auto-assigned'}
                      </p>
                    </div>
                    <div className="bg-gradient-to-br from-gray-50 to-white rounded-xl p-4 border border-gray-200">
                      <h4 className="font-semibold text-gray-900 mb-1">Date & Time</h4>
                      <p className="text-gray-700">
                        {selectedDate.toLocaleDateString()} at {selectedTime}
                      </p>
                    </div>
                    <div className="bg-gradient-to-br from-gray-50 to-white rounded-xl p-4 border border-gray-200">
                      <h4 className="font-semibold text-gray-900 mb-1">Duration</h4>
                      <p className="text-gray-700">{selectedPackage.durationMinutes} minutes</p>
                    </div>
                    <div className="bg-gradient-to-br from-emerald-50 to-teal-50 rounded-xl p-4 border border-emerald-200">
                      <h4 className="font-semibold text-emerald-900 mb-1">Price</h4>
                      <p className="text-emerald-700 font-bold text-2xl">₹{selectedPackage.price}</p>
                    </div>
                  </div>

                  {notes && (
                    <div className="bg-gradient-to-br from-blue-50 to-indigo-50 rounded-xl p-4 border border-blue-200">
                      <h4 className="font-semibold text-blue-900 mb-2">Notes</h4>
                      <p className="text-blue-800">{notes}</p>
                    </div>
                  )}

                  <div className="bg-gradient-to-r from-yellow-50 to-orange-50 p-6 rounded-2xl border border-yellow-200">
                    <h4 className="font-semibold text-yellow-900 mb-3 flex items-center">
                      <Shield className="h-5 w-5 mr-2" />
                      Please Note
                    </h4>
                    <ul className="text-sm text-yellow-800 space-y-2">
                      <li className="flex items-start">
                        <div className="w-2 h-2 bg-yellow-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                        This is a booking request, not a confirmed appointment
                      </li>
                      <li className="flex items-start">
                        <div className="w-2 h-2 bg-yellow-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                        The clinic will review and approve your request
                      </li>
                      <li className="flex items-start">
                        <div className="w-2 h-2 bg-yellow-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                        You'll receive a notification with the approval status
                      </li>
                      <li className="flex items-start">
                        <div className="w-2 h-2 bg-yellow-400 rounded-full mt-2 mr-3 flex-shrink-0"></div>
                        Once approved, you'll get final appointment confirmation
                      </li>
                    </ul>
                  </div>

                  <div className="flex gap-4 pt-4">
                    <Button 
                      variant="outline" 
                      onClick={() => setStep('datetime')} 
                      className="flex-1 border-2 border-gray-300 hover:border-emerald-300 hover:bg-emerald-50 py-3 rounded-xl font-semibold transition-all duration-300"
                    >
                      Go Back
                    </Button>
                    <Button 
                      onClick={handleBooking} 
                      className="flex-1 bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white py-3 rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl"
                    >
                      Submit Request
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}