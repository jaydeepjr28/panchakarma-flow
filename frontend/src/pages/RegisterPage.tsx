import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '@/context/AuthContext';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuInput } from '@/components/ui/NeuInput';
import {
  Mail,
  User,
  Phone,
  Eye,
  EyeOff,
  Sparkles,
  ArrowRight,
  ShieldCheck,
  CheckCircle,
  Building2,
  Activity,
  Heart,
} from 'lucide-react';

type RegistrationRole = 'patient' | 'clinic_admin' | 'therapist';

export const RegisterPage: React.FC = () => {
  const { register, isLoading } = useAuth();
  const navigate = useNavigate();

  const [selectedRole, setSelectedRole] = useState<RegistrationRole>('patient');
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [dateOfBirth, setDateOfBirth] = useState('1998-05-15');
  const [biologicalSex, setBiologicalSex] = useState('female');
  const [emergencyName, setEmergencyName] = useState('');
  const [emergencyPhone, setEmergencyPhone] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [agreeTerms, setAgreeTerms] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!agreeTerms) {
      setErrorMsg('Please agree to medical consent and terms to proceed.');
      return;
    }
    setErrorMsg('');
    try {
      const ok = await register({
        full_name: fullName,
        email,
        phone: phone || '+919876543210',
        password,
        date_of_birth: dateOfBirth || '1998-05-15',
        biological_sex: biologicalSex || 'female',
        emergency_contact_name: emergencyName.trim() || `${fullName}'s Emergency Contact`,
        emergency_contact_phone: emergencyPhone.trim() || phone || '+919876543211',
      });
      if (ok) {
        navigate('/patient/dashboard', { replace: true });
      }
    } catch (err: any) {
      setErrorMsg(err.message || 'Registration failed. Please check inputs.');
    }
  };

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col justify-center py-12 px-4 sm:px-6 lg:px-8">
      <div className="sm:mx-auto sm:w-full sm:max-w-md text-center mb-8">
        <Link to="/" className="inline-flex items-center gap-2 mb-2">
          <div className="w-12 h-12 rounded-2xl bg-[#F5F3EC] neu-raised flex items-center justify-center text-[#315C49]">
            <Sparkles className="w-6 h-6 text-[#C98769]" />
          </div>
        </Link>
        <h2 className="font-serif text-3xl font-bold text-[#234737] tracking-tight">
          Create Your Account
        </h2>
        <p className="mt-1 text-xs text-[#6F7B71]">
          Select your role below to get started with PanchakarmaFlow.
        </p>
      </div>

      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <NeuCard elevation="raised-lg" className="p-8">
          {/* ── Role Selector ── */}
          <div className="mb-6 pb-6 border-b border-[#E3DFC9]">
            <p className="text-[11px] uppercase tracking-wider font-bold text-[#6F7B71] mb-2.5">
              Register As
            </p>
            <div className="grid grid-cols-3 gap-2">
              {/* Patient — primary option */}
              <button
                type="button"
                onClick={() => setSelectedRole('patient')}
                className={`p-2.5 rounded-xl text-center text-xs transition-all flex flex-col items-center gap-1 ${
                  selectedRole === 'patient'
                    ? 'bg-[#315C49] text-white shadow-sm'
                    : 'bg-[#F5F3EC] text-[#27352D] neu-raised-sm hover:neu-pressed'
                }`}
              >
                <Heart className="w-4 h-4" />
                <span className="font-semibold">Patient</span>
              </button>

              {/* Clinic Admin — staff, requires invitation */}
              <button
                type="button"
                onClick={() => setSelectedRole('clinic_admin')}
                className={`p-2.5 rounded-xl text-center text-xs transition-all flex flex-col items-center gap-1 ${
                  selectedRole === 'clinic_admin'
                    ? 'bg-[#C98769] text-white shadow-sm'
                    : 'bg-[#F5F3EC] text-[#27352D] neu-raised-sm hover:neu-pressed'
                }`}
              >
                <Building2 className="w-4 h-4" />
                <span className="font-semibold">Clinic Admin</span>
              </button>

              {/* Therapist — staff, requires invitation */}
              <button
                type="button"
                onClick={() => setSelectedRole('therapist')}
                className={`p-2.5 rounded-xl text-center text-xs transition-all flex flex-col items-center gap-1 ${
                  selectedRole === 'therapist'
                    ? 'bg-[#357A55] text-white shadow-sm'
                    : 'bg-[#F5F3EC] text-[#27352D] neu-raised-sm hover:neu-pressed'
                }`}
              >
                <Activity className="w-4 h-4" />
                <span className="font-semibold">Therapist</span>
              </button>
            </div>

            {/* Context hint for staff roles */}
            {selectedRole !== 'patient' && (
              <p className="mt-2.5 text-[10px] text-[#C98769] bg-[#C98769]/10 border border-[#C98769]/20 rounded-lg px-3 py-2 leading-relaxed">
                ⚠️ Staff accounts require a clinic invitation link. Contact your clinic admin if you
                don't have one. Self-registration will create a patient account for that email.
              </p>
            )}

            {selectedRole === 'patient' && (
              <p className="mt-2.5 text-[10px] text-[#315C49] bg-[#315C49]/10 border border-[#315C49]/20 rounded-lg px-3 py-2 leading-relaxed">
                ✓ Patient accounts are open for self-registration. Fill in the form below to begin
                your Panchakarma therapy journey.
              </p>
            )}
          </div>
          {errorMsg && (
            <div className="mb-4 p-3 rounded-xl bg-[#B74747]/10 border border-[#B74747]/20 text-xs text-[#B74747]">
              {errorMsg}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <NeuInput
              label="Full Name"
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              placeholder="e.g. Meera Nambiar"
              leftIcon={<User className="w-4 h-4" />}
              required
            />

            <NeuInput
              label="Email Address"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="meera@example.com"
              leftIcon={<Mail className="w-4 h-4" />}
              required
            />

            <NeuInput
              label="Mobile Phone"
              type="tel"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="+91 98765 00000"
              leftIcon={<Phone className="w-4 h-4" />}
            />

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
                  Date of Birth
                </label>
                <input
                  type="date"
                  value={dateOfBirth}
                  onChange={(e) => setDateOfBirth(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none focus:ring-2 focus:ring-[#315C49]/40"
                  required
                />
              </div>
              <div>
                <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
                  Biological Sex
                </label>
                <select
                  value={biologicalSex}
                  onChange={(e) => setBiologicalSex(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none focus:ring-2 focus:ring-[#315C49]/40"
                >
                  <option value="female">Female</option>
                  <option value="male">Male</option>
                  <option value="other">Other</option>
                </select>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <NeuInput
                label="Emergency Contact"
                type="text"
                value={emergencyName}
                onChange={(e) => setEmergencyName(e.target.value)}
                placeholder="Name (e.g. Spouse/Parent)"
              />
              <NeuInput
                label="Contact Phone"
                type="tel"
                value={emergencyPhone}
                onChange={(e) => setEmergencyPhone(e.target.value)}
                placeholder="+91 98765 00001"
              />
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
                Create Secure Password
              </label>
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] placeholder-[#6F7B71]/60 focus:outline-none focus:ring-2 focus:ring-[#315C49]/40"
                  placeholder="Minimum 8 characters"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-2.5 text-[#6F7B71] hover:text-[#27352D]"
                >
                  {showPassword ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
            </div>

            <div className="pt-2">
              <label className="flex items-start gap-2.5 cursor-pointer text-xs text-[#6F7B71]">
                <input
                  type="checkbox"
                  checked={agreeTerms}
                  onChange={(e) => setAgreeTerms(e.target.checked)}
                  className="mt-0.5 rounded text-[#315C49] focus:ring-[#315C49]"
                />
                <span>
                  I consent to sharing baseline health data and following Ayurvedic pre-care protocols as prescribed.
                </span>
              </label>
            </div>

            <div className="pt-2">
              <NeuButton
                type="submit"
                variant="primary"
                size="md"
                isLoading={isLoading}
                className="w-full flex items-center justify-center gap-2"
              >
                Register & Open Portal
                <ArrowRight className="w-4 h-4" />
              </NeuButton>
            </div>
          </form>

          <div className="mt-6 pt-4 border-t border-[#E3DFC9] text-center text-xs text-[#6F7B71]">
            Already have an account?{' '}
            <Link to="/login" className="font-bold text-[#315C49] hover:underline">
              Sign In
            </Link>
          </div>
        </NeuCard>
      </div>
    </div>
  );
};
