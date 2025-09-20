import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Calendar } from '@/components/ui/calendar';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { ArrowLeft, Clock, Star, User, Calendar as CalendarIcon, CheckCircle } from 'lucide-react';
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
      <div className="min-h-screen bg-gray-50">
        {/* Header */}
        <div className="bg-white shadow-sm border-b">
          <div className="container mx-auto px-4 py-4">
            <div className="flex items-center">
              <Button variant="ghost" onClick={onBack} className="mr-4">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Clinics
              </Button>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">Booking Request Submitted</h1>
                <p className="text-gray-600">{clinic.name}</p>
              </div>
            </div>
          </div>
        </div>

        <div className="container mx-auto px-4 py-8">
          <div className="max-w-2xl mx-auto">
            <Card>
              <CardHeader className="text-center">
                <div className="w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-4">
                  <CheckCircle className="h-8 w-8 text-green-600" />
                </div>
                <CardTitle className="text-2xl">Request Submitted Successfully!</CardTitle>
                <CardDescription>
                  Your appointment request has been sent to {clinic.name} for approval.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="bg-blue-50 p-4 rounded-lg">
                  <h4 className="font-medium text-blue-900 mb-2">What happens next?</h4>
                  <ul className="text-sm text-blue-800 space-y-1">
                    <li>• The clinic will review your appointment request</li>
                    <li>• You'll receive a notification once it's approved or if any changes are needed</li>
                    <li>• Once approved, your appointment will be confirmed and assigned to a therapist</li>
                    <li>• You can track the status in your "My Appointments" section</li>
                  </ul>
                </div>

                <div className="border rounded-lg p-4">
                  <h4 className="font-medium text-gray-900 mb-3">Request Summary</h4>
                  <div className="grid md:grid-cols-2 gap-3 text-sm">
                    <div>
                      <span className="text-gray-600">Package:</span>
                      <p className="font-medium">{selectedPackage?.name}</p>
                    </div>
                    <div>
                      <span className="text-gray-600">Doctor:</span>
                      <p className="font-medium">
                        {selectedDoctor ? selectedDoctor.name : 'Auto-assigned'}
                      </p>
                    </div>
                    <div>
                      <span className="text-gray-600">Date & Time:</span>
                      <p className="font-medium">
                        {selectedDate?.toLocaleDateString()} at {selectedTime}
                      </p>
                    </div>
                    <div>
                      <span className="text-gray-600">Price:</span>
                      <p className="font-medium">₹{selectedPackage?.price}</p>
                    </div>
                  </div>
                </div>

                <div className="flex gap-4">
                  <Button onClick={onBookingComplete} className="flex-1">
                    Go to My Appointments
                  </Button>
                  <Button variant="outline" onClick={onBack} className="flex-1">
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
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center">
            <Button variant="ghost" onClick={onBack} className="mr-4">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Book Appointment</h1>
              <p className="text-gray-600">{clinic.name}</p>
            </div>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-8">
        {/* Progress Steps */}
        <div className="flex items-center justify-center mb-8">
          <div className="flex items-center space-x-4">
            {['package', 'doctor', 'datetime', 'confirm'].map((stepName, index) => (
              <div key={stepName} className="flex items-center">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
                  step === stepName ? 'bg-blue-600 text-white' : 
                  ['package', 'doctor', 'datetime', 'confirm'].indexOf(step) > index ? 'bg-green-600 text-white' : 'bg-gray-200 text-gray-600'
                }`}>
                  {index + 1}
                </div>
                {index < 3 && <div className="w-12 h-0.5 bg-gray-200 mx-2" />}
              </div>
            ))}
          </div>
        </div>

        {/* Step Content */}
        <div className="max-w-4xl mx-auto">
          {step === 'package' && (
            <div>
              <h2 className="text-xl font-semibold mb-6">Select Therapy Package</h2>
              <div className="grid md:grid-cols-2 gap-6">
                {packages.map((pkg) => (
                  <Card 
                    key={pkg.id} 
                    className={`cursor-pointer transition-all ${
                      selectedPackage?.id === pkg.id ? 'ring-2 ring-blue-600 bg-blue-50' : 'hover:shadow-lg'
                    }`}
                    onClick={() => setSelectedPackage(pkg)}
                  >
                    <CardHeader>
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <CardTitle className="text-lg">{pkg.name}</CardTitle>
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
                          {pkg.steps.slice(0, 3).map((step, index) => (
                            <div key={step.id} className="text-xs text-gray-600">
                              {index + 1}. {step.name}
                            </div>
                          ))}
                          {pkg.steps.length > 3 && (
                            <div className="text-xs text-gray-500">
                              +{pkg.steps.length - 3} more steps
                            </div>
                          )}
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
              
              {selectedPackage && (
                <div className="mt-6 text-center">
                  <Button onClick={() => setStep('doctor')}>
                    Continue to Doctor Selection
                  </Button>
                </div>
              )}
            </div>
          )}

          {step === 'doctor' && (
            <div>
              <h2 className="text-xl font-semibold mb-6">Select Doctor</h2>
              <div className="grid md:grid-cols-2 gap-6">
                <Card 
                  className={`cursor-pointer transition-all ${
                    selectedDoctor === null ? 'ring-2 ring-blue-600 bg-blue-50' : 'hover:shadow-lg'
                  }`}
                  onClick={() => setSelectedDoctor(null)}
                >
                  <CardHeader>
                    <CardTitle className="flex items-center">
                      <User className="h-5 w-5 mr-2" />
                      Auto-Assign Doctor
                    </CardTitle>
                    <CardDescription>
                      Let us assign the best available doctor for your appointment
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <p className="text-sm text-gray-600">
                      Our system will automatically assign a qualified doctor based on availability and expertise.
                    </p>
                  </CardContent>
                </Card>

                {doctors.map((doctor) => (
                  <Card 
                    key={doctor.id} 
                    className={`cursor-pointer transition-all ${
                      selectedDoctor?.id === doctor.id ? 'ring-2 ring-blue-600 bg-blue-50' : 'hover:shadow-lg'
                    }`}
                    onClick={() => setSelectedDoctor(doctor)}
                  >
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
                      <div className="space-y-2">
                        <div>
                          <span className="text-sm text-gray-600">Experience: </span>
                          <span className="font-medium">{doctor.experience} years</span>
                        </div>
                        <div>
                          <span className="text-sm text-gray-600">Specialties: </span>
                          <div className="flex flex-wrap gap-1 mt-1">
                            {doctor.specialties.map((specialty) => (
                              <Badge key={specialty} variant="secondary" className="text-xs">
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
              
              <div className="mt-6 text-center">
                <Button onClick={() => setStep('datetime')}>
                  Continue to Date & Time
                </Button>
              </div>
            </div>
          )}

          {step === 'datetime' && (
            <div>
              <h2 className="text-xl font-semibold mb-6">Select Date & Time</h2>
              <div className="grid lg:grid-cols-2 gap-8">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center">
                      <CalendarIcon className="h-5 w-5 mr-2" />
                      Select Date
                    </CardTitle>
                    <CardDescription>
                      Choose your preferred appointment date
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="flex justify-center">
                    <Calendar
                      mode="single"
                      selected={selectedDate}
                      onSelect={setSelectedDate}
                      disabled={isDateDisabled}
                      className="rounded-md border w-full max-w-sm"
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
                        day: "h-9 w-9 p-0 font-normal aria-selected:opacity-100",
                        day_selected: "bg-primary text-primary-foreground hover:bg-primary hover:text-primary-foreground focus:bg-primary focus:text-primary-foreground",
                        day_today: "bg-accent text-accent-foreground",
                        day_outside: "text-muted-foreground opacity-50",
                        day_disabled: "text-muted-foreground opacity-50",
                        day_range_middle: "aria-selected:bg-accent aria-selected:text-accent-foreground",
                        day_hidden: "invisible",
                      }}
                    />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center">
                      <Clock className="h-5 w-5 mr-2" />
                      Select Time
                    </CardTitle>
                    <CardDescription>
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
                              className="text-sm h-10"
                              onClick={() => setSelectedTime(slot)}
                            >
                              {slot}
                            </Button>
                          ))}
                        </div>
                      ) : (
                        <div className="text-center py-8">
                          <Clock className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                          <p className="text-gray-600 mb-2">No available slots for this date.</p>
                          <p className="text-sm text-gray-500">Please select another date or try auto-assigning a doctor.</p>
                        </div>
                      )
                    ) : (
                      <div className="text-center py-8">
                        <CalendarIcon className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                        <p className="text-gray-600">Please select a date first.</p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </div>

              <Card className="mt-6">
                <CardHeader>
                  <CardTitle>Additional Notes (Optional)</CardTitle>
                  <CardDescription>
                    Any specific requirements, health conditions, or preferences
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <Textarea
                    placeholder="e.g., I have back pain, prefer morning sessions, allergic to sesame oil..."
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    rows={3}
                  />
                </CardContent>
              </Card>
              
              {selectedDate && selectedTime && (
                <div className="mt-6 text-center">
                  <Button onClick={() => setStep('confirm')} size="lg">
                    Review Booking Request
                  </Button>
                </div>
              )}
            </div>
          )}

          {step === 'confirm' && selectedPackage && selectedDate && selectedTime && (
            <div>
              <h2 className="text-xl font-semibold mb-6">Confirm Your Booking Request</h2>
              <Card>
                <CardHeader>
                  <CardTitle>Request Summary</CardTitle>
                  <CardDescription>
                    This will be sent to the clinic for approval. You'll be notified once it's reviewed.
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid md:grid-cols-2 gap-4">
                    <div>
                      <h4 className="font-medium text-gray-900">Clinic</h4>
                      <p className="text-gray-600">{clinic.name}</p>
                    </div>
                    <div>
                      <h4 className="font-medium text-gray-900">Package</h4>
                      <p className="text-gray-600">{selectedPackage.name}</p>
                    </div>
                    <div>
                      <h4 className="font-medium text-gray-900">Doctor</h4>
                      <p className="text-gray-600">
                        {selectedDoctor ? selectedDoctor.name : 'Auto-assigned'}
                      </p>
                    </div>
                    <div>
                      <h4 className="font-medium text-gray-900">Date & Time</h4>
                      <p className="text-gray-600">
                        {selectedDate.toLocaleDateString()} at {selectedTime}
                      </p>
                    </div>
                    <div>
                      <h4 className="font-medium text-gray-900">Duration</h4>
                      <p className="text-gray-600">{selectedPackage.durationMinutes} minutes</p>
                    </div>
                    <div>
                      <h4 className="font-medium text-gray-900">Price</h4>
                      <p className="text-gray-600 font-bold text-lg">₹{selectedPackage.price}</p>
                    </div>
                  </div>

                  {notes && (
                    <div>
                      <h4 className="font-medium text-gray-900">Notes</h4>
                      <p className="text-gray-600">{notes}</p>
                    </div>
                  )}

                  <div className="bg-yellow-50 p-4 rounded-lg">
                    <h4 className="font-medium text-yellow-900 mb-2">Please Note</h4>
                    <ul className="text-sm text-yellow-800 space-y-1">
                      <li>• This is a booking request, not a confirmed appointment</li>
                      <li>• The clinic will review and approve your request</li>
                      <li>• You'll receive a notification with the approval status</li>
                      <li>• Once approved, you'll get final appointment confirmation</li>
                    </ul>
                  </div>

                  <div className="flex gap-4 pt-4">
                    <Button variant="outline" onClick={() => setStep('datetime')} className="flex-1">
                      Go Back
                    </Button>
                    <Button onClick={handleBooking} className="flex-1">
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