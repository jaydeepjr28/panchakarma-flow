import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Heart, Users, Calendar, Activity } from 'lucide-react';
import { initializeData, getCurrentUser, setCurrentUser, getStoredData, setStoredData, generateId } from '@/lib/data';
import { User, UserRole } from '@/lib/types';
import PatientDashboard from './PatientDashboard';
import ClinicDashboard from './ClinicDashboard';
import TherapistDashboard from './TherapistDashboard';

export default function Index() {
  const [currentUser, setCurrentUserState] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    initializeData();
    const user = getCurrentUser();
    setCurrentUserState(user);
    setIsLoading(false);
  }, []);

  const handleLogin = (email: string, role: UserRole) => {
    const users = getStoredData('users', []);
    let user = users.find(u => u.email === email && u.role === role);
    
    if (!user) {
      // Create new user for demo purposes
      user = {
        id: generateId(),
        name: email.split('@')[0],
        email,
        phone: '+1234567890',
        role,
        clinicId: role !== 'patient' ? 'clinic1' : undefined
      };
      users.push(user);
      setStoredData('users', users);
    }
    
    setCurrentUser(user.id);
    setCurrentUserState(user);
  };

  const handleLogout = () => {
    localStorage.removeItem('currentUserId');
    setCurrentUserState(null);
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (currentUser) {
    switch (currentUser.role) {
      case 'patient':
        return <PatientDashboard user={currentUser} onLogout={handleLogout} />;
      case 'clinic_admin':
        return <ClinicDashboard user={currentUser} onLogout={handleLogout} />;
      case 'therapist':
        return <TherapistDashboard user={currentUser} onLogout={handleLogout} />;
      default:
        return <LoginScreen onLogin={handleLogin} />;
    }
  }

  return <LoginScreen onLogin={handleLogin} />;
}

function LoginScreen({ onLogin }: { onLogin: (email: string, role: UserRole) => void }) {
  const [email, setEmail] = useState('');
  const [selectedRole, setSelectedRole] = useState<UserRole>('patient');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (email) {
      onLogin(email, selectedRole);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-green-50">
      {/* Hero Section */}
      <div className="container mx-auto px-4 py-12">
        <div className="text-center mb-12">
          <div className="flex items-center justify-center mb-6">
            <Heart className="h-12 w-12 text-green-600 mr-3" />
            <h1 className="text-4xl font-bold text-gray-900">PanchakarmaFlow</h1>
          </div>
          <p className="text-xl text-gray-600 max-w-2xl mx-auto">
            Complete Panchakarma therapy management system connecting patients with authentic Ayurvedic practitioners
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid md:grid-cols-3 gap-8 mb-12">
          <Card className="text-center hover:shadow-lg transition-shadow">
            <CardHeader>
              <Users className="h-12 w-12 text-blue-600 mx-auto mb-4" />
              <CardTitle>For Patients</CardTitle>
              <CardDescription>
                Browse clinics, book appointments, and track your Panchakarma journey
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="text-center hover:shadow-lg transition-shadow">
            <CardHeader>
              <Calendar className="h-12 w-12 text-green-600 mx-auto mb-4" />
              <CardTitle>For Clinics</CardTitle>
              <CardDescription>
                Manage doctors, schedules, and therapy packages efficiently
              </CardDescription>
            </CardHeader>
          </Card>

          <Card className="text-center hover:shadow-lg transition-shadow">
            <CardHeader>
              <Activity className="h-12 w-12 text-purple-600 mx-auto mb-4" />
              <CardTitle>For Therapists</CardTitle>
              <CardDescription>
                Track therapy sessions step-by-step and monitor patient progress
              </CardDescription>
            </CardHeader>
          </Card>
        </div>

        {/* Login Section */}
        <div className="max-w-md mx-auto">
          <Card>
            <CardHeader>
              <CardTitle>Access Your Dashboard</CardTitle>
              <CardDescription>
                Enter your email and select your role to continue
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder="Enter your email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="role">Role</Label>
                  <Select value={selectedRole} onValueChange={(value) => setSelectedRole(value as UserRole)}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select your role" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="patient">Patient</SelectItem>
                      <SelectItem value="clinic_admin">Clinic Administrator</SelectItem>
                      <SelectItem value="therapist">Therapist</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <Button type="submit" className="w-full">
                  Access Dashboard
                </Button>
              </form>

              <div className="mt-6 p-4 bg-gray-50 rounded-lg">
                <h4 className="font-semibold text-sm mb-2">Demo Accounts:</h4>
                <div className="text-xs space-y-1 text-gray-600">
                  <p><strong>Patient:</strong> john@example.com</p>
                  <p><strong>Clinic Admin:</strong> sarah@ayurclinic.com</p>
                  <p><strong>Therapist:</strong> raj@ayurclinic.com</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}