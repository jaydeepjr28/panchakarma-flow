import { useState, useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Heart, Users, Calendar, Activity, Mail, Lock, Sparkles, Shield, CheckCircle, ArrowRight, Eye, EyeOff } from 'lucide-react';
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
      <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-teal-50 to-cyan-50 flex items-center justify-center">
        <div className="text-center">
          <div className="relative w-20 h-20 mx-auto mb-4">
            <div className="absolute inset-0 bg-gradient-to-r from-emerald-400 to-teal-500 rounded-full animate-spin">
              <div className="absolute inset-2 bg-white rounded-full"></div>
            </div>
            <Heart className="absolute inset-0 w-8 h-8 m-auto text-emerald-500" />
          </div>
          <p className="text-gray-600 font-medium">Loading PanchakarmaFlow...</p>
        </div>
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
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (email) {
      setIsLoading(true);
      // Simulate loading for better UX
      await new Promise(resolve => setTimeout(resolve, 800));
      onLogin(email, selectedRole);
      setIsLoading(false);
    }
  };

  const quickLogin = (demoEmail: string, role: UserRole) => {
    setEmail(demoEmail);
    setSelectedRole(role);
    setTimeout(() => onLogin(demoEmail, role), 300);
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-teal-50 to-cyan-50 relative overflow-hidden">
      {/* Background decorative elements */}
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-gradient-to-br from-emerald-200/30 to-teal-300/30 rounded-full blur-3xl"></div>
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-gradient-to-tr from-cyan-200/30 to-blue-300/30 rounded-full blur-3xl"></div>
        <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-gradient-to-r from-teal-100/20 to-emerald-100/20 rounded-full blur-3xl"></div>
      </div>

      <div className="relative z-10">
        {/* Hero Section */}
        <div className="container mx-auto px-4 py-12">
          <div className="text-center mb-16 animate-in fade-in duration-1000">
            <div className="flex items-center justify-center mb-8">
              <div className="relative">
                <div className="w-16 h-16 bg-gradient-to-br from-emerald-500 to-teal-600 rounded-2xl flex items-center justify-center shadow-2xl shadow-emerald-500/25">
                  <Heart className="h-8 w-8 text-white" />
                </div>
                <div className="absolute -top-1 -right-1 w-6 h-6 bg-gradient-to-r from-yellow-400 to-orange-400 rounded-full flex items-center justify-center">
                  <Sparkles className="h-3 w-3 text-white" />
                </div>
              </div>
              <div className="ml-4">
                <h1 className="text-5xl font-bold bg-gradient-to-r from-emerald-600 via-teal-600 to-cyan-600 bg-clip-text text-transparent">
                  PanchakarmaFlow
                </h1>
                <div className="flex items-center justify-center mt-2">
                  <div className="w-12 h-0.5 bg-gradient-to-r from-emerald-400 to-teal-500 rounded-full"></div>
                  <Sparkles className="h-3 w-3 text-emerald-500 mx-2" />
                  <div className="w-12 h-0.5 bg-gradient-to-r from-teal-500 to-cyan-400 rounded-full"></div>
                </div>
              </div>
            </div>
            <p className="text-xl text-gray-700 max-w-3xl mx-auto leading-relaxed">
              Complete Panchakarma therapy management system connecting patients with 
              <span className="font-semibold text-emerald-600"> authentic Ayurvedic practitioners</span>
            </p>
            <div className="flex items-center justify-center mt-6 space-x-6">
              <div className="flex items-center text-sm text-gray-600">
                <Shield className="h-4 w-4 text-emerald-500 mr-2" />
                Secure & Private
              </div>
              <div className="flex items-center text-sm text-gray-600">
                <CheckCircle className="h-4 w-4 text-emerald-500 mr-2" />
                Verified Practitioners
              </div>
              <div className="flex items-center text-sm text-gray-600">
                <Heart className="h-4 w-4 text-emerald-500 mr-2" />
                Holistic Care
              </div>
            </div>
          </div>

          {/* Features Grid */}
          <div className="grid md:grid-cols-3 gap-8 mb-16 animate-in slide-in-from-bottom duration-1000">
            <Card className="text-center hover:shadow-2xl transition-all duration-500 hover:-translate-y-2 border-0 bg-white/80 backdrop-blur-sm">
              <CardHeader className="pb-4">
                <div className="w-16 h-16 bg-gradient-to-br from-blue-400 to-blue-600 rounded-2xl flex items-center justify-center mx-auto mb-4 shadow-lg shadow-blue-500/25">
                  <Users className="h-8 w-8 text-white" />
                </div>
                <CardTitle className="text-xl font-bold text-gray-800">For Patients</CardTitle>
                <CardDescription className="text-gray-600 leading-relaxed">
                  Browse clinics, book appointments, and track your Panchakarma journey with ease
                </CardDescription>
              </CardHeader>
            </Card>

            <Card className="text-center hover:shadow-2xl transition-all duration-500 hover:-translate-y-2 border-0 bg-white/80 backdrop-blur-sm">
              <CardHeader className="pb-4">
                <div className="w-16 h-16 bg-gradient-to-br from-emerald-400 to-emerald-600 rounded-2xl flex items-center justify-center mx-auto mb-4 shadow-lg shadow-emerald-500/25">
                  <Calendar className="h-8 w-8 text-white" />
                </div>
                <CardTitle className="text-xl font-bold text-gray-800">For Clinics</CardTitle>
                <CardDescription className="text-gray-600 leading-relaxed">
                  Manage doctors, schedules, and therapy packages efficiently with our comprehensive tools
                </CardDescription>
              </CardHeader>
            </Card>

            <Card className="text-center hover:shadow-2xl transition-all duration-500 hover:-translate-y-2 border-0 bg-white/80 backdrop-blur-sm">
              <CardHeader className="pb-4">
                <div className="w-16 h-16 bg-gradient-to-br from-purple-400 to-purple-600 rounded-2xl flex items-center justify-center mx-auto mb-4 shadow-lg shadow-purple-500/25">
                  <Activity className="h-8 w-8 text-white" />
                </div>
                <CardTitle className="text-xl font-bold text-gray-800">For Therapists</CardTitle>
                <CardDescription className="text-gray-600 leading-relaxed">
                  Track therapy sessions step-by-step and monitor patient progress in real-time
                </CardDescription>
              </CardHeader>
            </Card>
          </div>

          {/* Login Section */}
          <div className="max-w-md mx-auto animate-in scale-in duration-1000">
            <Card className="border-0 shadow-2xl bg-white/90 backdrop-blur-sm">
              <CardHeader className="text-center pb-6">
                <CardTitle className="text-2xl font-bold text-gray-800 flex items-center justify-center">
                  <Lock className="h-6 w-6 text-emerald-600 mr-2" />
                  Access Your Dashboard
                </CardTitle>
                <CardDescription className="text-gray-600 mt-2">
                  Enter your credentials to continue your wellness journey
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleSubmit} className="space-y-6">
                  <div className="space-y-2">
                    <Label htmlFor="email" className="text-sm font-semibold text-gray-700">
                      Email Address
                    </Label>
                    <div className="relative">
                      <Mail className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
                      <Input
                        id="email"
                        type="email"
                        placeholder="Enter your email address"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        className="pl-10 h-12 border-2 border-gray-200 rounded-xl focus:border-emerald-300 focus:ring-emerald-200 transition-all duration-300"
                        required
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="role" className="text-sm font-semibold text-gray-700">
                      Select Your Role
                    </Label>
                    <Select value={selectedRole} onValueChange={(value) => setSelectedRole(value as UserRole)}>
                      <SelectTrigger className="h-12 border-2 border-gray-200 rounded-xl focus:border-emerald-300 focus:ring-emerald-200 transition-all duration-300">
                        <SelectValue placeholder="Choose your role" />
                      </SelectTrigger>
                      <SelectContent className="border-2 border-gray-100 rounded-xl">
                        <SelectItem value="patient" className="rounded-lg">
                          <div className="flex items-center">
                            <Users className="h-4 w-4 text-blue-500 mr-2" />
                            Patient
                          </div>
                        </SelectItem>
                        <SelectItem value="clinic_admin" className="rounded-lg">
                          <div className="flex items-center">
                            <Calendar className="h-4 w-4 text-emerald-500 mr-2" />
                            Clinic Administrator
                          </div>
                        </SelectItem>
                        <SelectItem value="therapist" className="rounded-lg">
                          <div className="flex items-center">
                            <Activity className="h-4 w-4 text-purple-500 mr-2" />
                            Therapist
                          </div>
                        </SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <Button 
                    type="submit" 
                    className="w-full h-12 bg-gradient-to-r from-emerald-500 to-teal-600 hover:from-emerald-600 hover:to-teal-700 text-white font-semibold rounded-xl transition-all duration-300 shadow-lg hover:shadow-xl disabled:opacity-50"
                    disabled={isLoading}
                  >
                    {isLoading ? (
                      <div className="flex items-center">
                        <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin mr-2"></div>
                        Signing In...
                      </div>
                    ) : (
                      <div className="flex items-center">
                        Access Dashboard
                        <ArrowRight className="h-5 w-5 ml-2" />
                      </div>
                    )}
                  </Button>
                </form>

                <div className="mt-8 p-6 bg-gradient-to-r from-gray-50 to-emerald-50 rounded-2xl border border-gray-100">
                  <h4 className="font-semibold text-sm mb-4 text-gray-800 flex items-center">
                    <Sparkles className="h-4 w-4 text-emerald-500 mr-2" />
                    Quick Demo Access
                  </h4>
                  <div className="space-y-3">
                    <button
                      onClick={() => quickLogin('sarah@ayurclinic.com', 'clinic_admin')}
                      className="w-full text-left p-3 bg-white rounded-xl border border-gray-200 hover:border-emerald-300 hover:bg-emerald-50 transition-all duration-300 group"
                    >
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="font-medium text-gray-800 group-hover:text-emerald-700">Clinic Admin Demo</p>
                          <p className="text-xs text-gray-600">sarah@ayurclinic.com</p>
                        </div>
                        <Calendar className="h-4 w-4 text-emerald-500" />
                      </div>
                    </button>
                    <button
                      onClick={() => quickLogin('raj@ayurclinic.com', 'therapist')}
                      className="w-full text-left p-3 bg-white rounded-xl border border-gray-200 hover:border-emerald-300 hover:bg-emerald-50 transition-all duration-300 group"
                    >
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="font-medium text-gray-800 group-hover:text-emerald-700">Therapist Demo</p>
                          <p className="text-xs text-gray-600">raj@ayurclinic.com</p>
                        </div>
                        <Activity className="h-4 w-4 text-purple-500" />
                      </div>
                    </button>
                  </div>
                </div>

                <div className="mt-6 text-center">
                  <p className="text-xs text-gray-500">
                    By continuing, you agree to our terms of service and privacy policy
                  </p>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Footer */}
          <div className="text-center mt-16 animate-in fade-in duration-1000">
            <p className="text-gray-600 mb-4">
              Trusted by healthcare professionals across the country
            </p>
            <div className="flex items-center justify-center space-x-8 text-sm text-gray-500">
              <div className="flex items-center">
                <Shield className="h-4 w-4 mr-1" />
                HIPAA Compliant
              </div>
              <div className="flex items-center">
                <CheckCircle className="h-4 w-4 mr-1" />
                ISO Certified
              </div>
              <div className="flex items-center">
                <Heart className="h-4 w-4 mr-1" />
                24/7 Support
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
