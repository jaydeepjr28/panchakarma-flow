import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Progress } from '@/components/ui/progress';
import { ArrowLeft, Play, Pause, CheckCircle, Clock, User, Star } from 'lucide-react';
import { Appointment, Doctor, Session, SessionStep, TherapyPackage, User as UserType } from '@/lib/types';
import { getStoredData, setStoredData, generateId } from '@/lib/data';

interface SessionTrackerProps {
  appointment: Appointment;
  doctor: Doctor;
  onBack: () => void;
  onSessionComplete: () => void;
}

export default function SessionTracker({ appointment, doctor, onBack, onSessionComplete }: SessionTrackerProps) {
  const [session, setSession] = useState<Session | null>(null);
  const [therapyPackage, setTherapyPackage] = useState<TherapyPackage | null>(null);
  const [patient, setPatient] = useState<UserType | null>(null);
  const [currentStepNotes, setCurrentStepNotes] = useState<string>('');
  const [sessionNotes, setSessionNotes] = useState<string>('');
  const [isStepActive, setIsStepActive] = useState<boolean>(false);
  const [activeStepStartTime, setActiveStepStartTime] = useState<string | null>(null);

  useEffect(() => {
    // Load therapy package
    const packages = getStoredData('packages', []);
    const pkg = packages.find(p => p.id === appointment.packageId);
    setTherapyPackage(pkg || null);

    // Load patient
    const users = getStoredData('users', []);
    const patientUser = users.find(u => u.id === appointment.patientId);
    setPatient(patientUser || null);

    // Load or create session
    const sessions = getStoredData('sessions', []);
    let existingSession = sessions.find(s => s.appointmentId === appointment.id);
    
    if (!existingSession && pkg) {
      // Create new session
      existingSession = {
        id: generateId(),
        appointmentId: appointment.id,
        currentStep: 0,
        completedSteps: [],
        therapistNotes: '',
      };
      sessions.push(existingSession);
      setStoredData('sessions', sessions);
    }
    
    setSession(existingSession || null);
    setSessionNotes(existingSession?.therapistNotes || '');
  }, [appointment]);

  const startSession = () => {
    if (!session) return;

    const updatedSession = {
      ...session,
      startTime: new Date().toISOString()
    };

    // Update session in storage
    const sessions = getStoredData('sessions', []);
    const updatedSessions = sessions.map(s => 
      s.id === session.id ? updatedSession : s
    );
    setStoredData('sessions', updatedSessions);

    // Update appointment status
    const appointments = getStoredData('appointments', []);
    const updatedAppointments = appointments.map(apt => 
      apt.id === appointment.id ? { ...apt, status: 'in_progress' as const } : apt
    );
    setStoredData('appointments', updatedAppointments);

    setSession(updatedSession);
  };

  const startStep = (stepIndex: number) => {
    setIsStepActive(true);
    setActiveStepStartTime(new Date().toISOString());
  };

  const completeStep = (stepIndex: number) => {
    if (!session || !therapyPackage) return;

    const step = therapyPackage.steps[stepIndex];
    const completedStep: SessionStep = {
      stepId: step.id,
      completed: true,
      startTime: activeStepStartTime || new Date().toISOString(),
      endTime: new Date().toISOString(),
      notes: currentStepNotes
    };

    const updatedSession = {
      ...session,
      currentStep: stepIndex + 1,
      completedSteps: [...session.completedSteps, completedStep],
      therapistNotes: sessionNotes
    };

    // Update session in storage
    const sessions = getStoredData('sessions', []);
    const updatedSessions = sessions.map(s => 
      s.id === session.id ? updatedSession : s
    );
    setStoredData('sessions', updatedSessions);

    setSession(updatedSession);
    setCurrentStepNotes('');
    setIsStepActive(false);
    setActiveStepStartTime(null);
  };

  const completeSession = () => {
    if (!session) return;

    const updatedSession = {
      ...session,
      endTime: new Date().toISOString(),
      therapistNotes: sessionNotes
    };

    // Update session in storage
    const sessions = getStoredData('sessions', []);
    const updatedSessions = sessions.map(s => 
      s.id === session.id ? updatedSession : s
    );
    setStoredData('sessions', updatedSessions);

    // Update appointment status
    const appointments = getStoredData('appointments', []);
    const updatedAppointments = appointments.map(apt => 
      apt.id === appointment.id ? { ...apt, status: 'completed' as const } : apt
    );
    setStoredData('appointments', updatedAppointments);

    // Create notification for patient
    const notifications = getStoredData('notifications', []);
    notifications.push({
      id: generateId(),
      userId: appointment.patientId,
      title: 'Session Completed',
      message: `Your ${therapyPackage?.name} session has been completed. Please follow the post-care instructions.`,
      type: 'success',
      read: false,
      createdAt: new Date().toISOString()
    });
    setStoredData('notifications', notifications);

    onSessionComplete();
  };

  if (!session || !therapyPackage || !patient) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <p className="text-gray-600">Loading session information...</p>
      </div>
    );
  }

  const progress = (session.completedSteps.length / therapyPackage.steps.length) * 100;
  const isSessionStarted = !!session.startTime;
  const isSessionCompleted = session.completedSteps.length === therapyPackage.steps.length;

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="container mx-auto px-4 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <Button variant="ghost" onClick={onBack} className="mr-4">
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back
              </Button>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">Session Tracker</h1>
                <p className="text-gray-600">{therapyPackage.name} - {patient.name}</p>
              </div>
            </div>
            <div className="text-right">
              <p className="text-sm text-gray-600">
                {new Date(appointment.date).toLocaleDateString()} at {appointment.time}
              </p>
              <Badge className={
                appointment.status === 'completed' ? 'bg-green-100 text-green-800' : 
                appointment.status === 'in_progress' ? 'bg-purple-100 text-purple-800' : 
                'bg-blue-100 text-blue-800'
              }>
                {appointment.status.replace('_', ' ').toUpperCase()}
              </Badge>
            </div>
          </div>
        </div>
      </div>

      <div className="container mx-auto px-4 py-8">
        <div className="grid lg:grid-cols-3 gap-8">
          {/* Patient Info & Progress */}
          <div className="lg:col-span-1 space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center">
                  <User className="h-5 w-5 mr-2" />
                  Patient Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <label className="text-sm font-medium text-gray-700">Name</label>
                  <p className="text-gray-900">{patient.name}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700">Contact</label>
                  <p className="text-gray-900">{patient.phone}</p>
                </div>
                <div>
                  <label className="text-sm font-medium text-gray-700">Email</label>
                  <p className="text-gray-900">{patient.email}</p>
                </div>
                {appointment.notes && (
                  <div>
                    <label className="text-sm font-medium text-gray-700">Patient Notes</label>
                    <p className="text-gray-900 text-sm bg-yellow-50 p-2 rounded">{appointment.notes}</p>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Session Progress</CardTitle>
                <CardDescription>
                  {session.completedSteps.length} of {therapyPackage.steps.length} steps completed
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Progress value={progress} className="mb-4" />
                <div className="space-y-2">
                  {therapyPackage.steps.map((step, index) => {
                    const isCompleted = session.completedSteps.some(cs => cs.stepId === step.id);
                    const isCurrent = index === session.currentStep && !isCompleted;
                    
                    return (
                      <div key={step.id} className={`flex items-center p-2 rounded ${
                        isCompleted ? 'bg-green-50' : 
                        isCurrent ? 'bg-blue-50' : 'bg-gray-50'
                      }`}>
                        <div className={`w-6 h-6 rounded-full flex items-center justify-center mr-3 ${
                          isCompleted ? 'bg-green-600 text-white' : 
                          isCurrent ? 'bg-blue-600 text-white' : 'bg-gray-300 text-gray-600'
                        }`}>
                          {isCompleted ? <CheckCircle className="h-4 w-4" /> : index + 1}
                        </div>
                        <div className="flex-1">
                          <p className={`text-sm font-medium ${
                            isCompleted ? 'text-green-800' : 
                            isCurrent ? 'text-blue-800' : 'text-gray-700'
                          }`}>
                            {step.name}
                          </p>
                          <p className="text-xs text-gray-600">{step.durationMinutes} min</p>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </CardContent>
            </Card>

            {/* Session Controls */}
            <Card>
              <CardHeader>
                <CardTitle>Session Controls</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {!isSessionStarted ? (
                  <Button onClick={startSession} className="w-full">
                    <Play className="h-4 w-4 mr-2" />
                    Start Session
                  </Button>
                ) : isSessionCompleted ? (
                  <Button onClick={completeSession} className="w-full bg-green-600 hover:bg-green-700">
                    <CheckCircle className="h-4 w-4 mr-2" />
                    Complete Session
                  </Button>
                ) : (
                  <div className="space-y-2">
                    <div className="flex items-center text-sm text-gray-600">
                      <Clock className="h-4 w-4 mr-2" />
                      Session started at {new Date(session.startTime!).toLocaleTimeString()}
                    </div>
                    <p className="text-sm text-green-600 font-medium">Session in progress...</p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>

          {/* Current Step Details */}
          <div className="lg:col-span-2 space-y-6">
            {isSessionStarted && !isSessionCompleted && (
              <Card>
                <CardHeader>
                  <CardTitle>Current Step: {therapyPackage.steps[session.currentStep]?.name}</CardTitle>
                  <CardDescription>
                    Step {session.currentStep + 1} of {therapyPackage.steps.length} • 
                    Duration: {therapyPackage.steps[session.currentStep]?.durationMinutes} minutes
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div>
                    <h4 className="font-medium mb-2">Description</h4>
                    <p className="text-gray-700">{therapyPackage.steps[session.currentStep]?.description}</p>
                  </div>

                  <div>
                    <h4 className="font-medium mb-2">Instructions</h4>
                    <ul className="list-disc list-inside space-y-1 text-gray-700">
                      {therapyPackage.steps[session.currentStep]?.instructions.map((instruction, index) => (
                        <li key={index} className="text-sm">{instruction}</li>
                      ))}
                    </ul>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Step Notes
                    </label>
                    <Textarea
                      placeholder="Record observations, patient response, any modifications made..."
                      value={currentStepNotes}
                      onChange={(e) => setCurrentStepNotes(e.target.value)}
                      rows={3}
                    />
                  </div>

                  <div className="flex gap-3">
                    {!isStepActive ? (
                      <Button 
                        onClick={() => startStep(session.currentStep)}
                        className="flex-1"
                      >
                        <Play className="h-4 w-4 mr-2" />
                        Start This Step
                      </Button>
                    ) : (
                      <Button 
                        onClick={() => completeStep(session.currentStep)}
                        className="flex-1 bg-green-600 hover:bg-green-700"
                      >
                        <CheckCircle className="h-4 w-4 mr-2" />
                        Complete Step
                      </Button>
                    )}
                  </div>

                  {isStepActive && (
                    <div className="bg-blue-50 p-3 rounded-lg">
                      <p className="text-sm text-blue-800">
                        <strong>Step started at:</strong> {new Date(activeStepStartTime!).toLocaleTimeString()}
                      </p>
                    </div>
                  )}
                </CardContent>
              </Card>
            )}

            {/* Completed Steps */}
            {session.completedSteps.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle>Completed Steps</CardTitle>
                  <CardDescription>Review of completed procedures</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {session.completedSteps.map((completedStep, index) => {
                      const step = therapyPackage.steps.find(s => s.id === completedStep.stepId);
                      return (
                        <div key={completedStep.stepId} className="border-l-4 border-green-500 pl-4 py-2">
                          <div className="flex items-center justify-between mb-2">
                            <h4 className="font-medium text-green-800">{step?.name}</h4>
                            <div className="text-sm text-gray-600">
                              {completedStep.startTime && completedStep.endTime && (
                                <>
                                  {new Date(completedStep.startTime).toLocaleTimeString()} - 
                                  {new Date(completedStep.endTime).toLocaleTimeString()}
                                </>
                              )}
                            </div>
                          </div>
                          {completedStep.notes && (
                            <p className="text-sm text-gray-700 bg-gray-50 p-2 rounded">
                              {completedStep.notes}
                            </p>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Session Notes */}
            <Card>
              <CardHeader>
                <CardTitle>Session Notes</CardTitle>
                <CardDescription>Overall observations and recommendations</CardDescription>
              </CardHeader>
              <CardContent>
                <Textarea
                  placeholder="Record overall session notes, patient response, recommendations for future sessions..."
                  value={sessionNotes}
                  onChange={(e) => setSessionNotes(e.target.value)}
                  rows={4}
                />
                <Button 
                  variant="outline" 
                  className="mt-3"
                  onClick={() => {
                    if (session) {
                      const sessions = getStoredData('sessions', []);
                      const updatedSessions = sessions.map(s => 
                        s.id === session.id ? { ...s, therapistNotes: sessionNotes } : s
                      );
                      setStoredData('sessions', updatedSessions);
                    }
                  }}
                >
                  Save Notes
                </Button>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}