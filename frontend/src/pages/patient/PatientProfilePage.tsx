import React, { useState } from 'react';
import { useAuth } from '@/context/AuthContext';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuInput } from '@/components/ui/NeuInput';
import { NeuBadge } from '@/components/ui/NeuBadge';
import {
  User,
  Heart,
  AlertTriangle,
  Phone,
  Home,
  Save,
  CheckCircle2,
  Sparkles,
} from 'lucide-react';

export const PatientProfilePage: React.FC = () => {
  const { user, profile, updateProfile } = useAuth();

  const [fullName, setFullName] = useState(profile?.full_name || user?.email?.split('@')[0] || '');
  const [bloodGroup, setBloodGroup] = useState(profile?.blood_group || 'O+');
  const [prakritiType, setPrakritiType] = useState(profile?.prakriti_type || 'Tridoshic');
  const [emergencyName, setEmergencyName] = useState(profile?.emergency_contact_name || '');
  const [emergencyPhone, setEmergencyPhone] = useState(profile?.emergency_contact_phone || '');
  const [allergiesInput, setAllergiesInput] = useState(profile?.allergies?.join(', ') || '');
  const [medicalHistory, setMedicalHistory] = useState(profile?.medical_history || '');

  const [isSaved, setIsSaved] = useState(false);

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    updateProfile({
      full_name: fullName,
      blood_group: bloodGroup,
      prakriti_type: prakritiType,
      emergency_contact_name: emergencyName,
      emergency_contact_phone: emergencyPhone,
      allergies: allergiesInput.split(',').map((s) => s.trim()).filter(Boolean),
      medical_history: medicalHistory,
    });
    setIsSaved(true);
    setTimeout(() => setIsSaved(false), 3000);
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div>
        <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
          Health Records & Vitals
        </p>
        <h1 className="font-serif text-3xl font-bold text-[#234737]">
          Patient Health Profile
        </h1>
        <p className="text-xs text-[#6F7B71]">
          Manage your biological constitution, known allergies, and emergency clinical contacts.
        </p>
      </div>

      {isSaved && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 shrink-0" />
          <span>Patient medical profile updated successfully.</span>
        </div>
      )}

      <form onSubmit={handleSave} className="space-y-6">
        {/* Core Personal Details */}
        <NeuCard elevation="raised" className="p-6 space-y-4">
          <h2 className="font-serif text-lg font-bold text-[#234737] border-b border-[#E3DFC9] pb-3">
            Personal & Ayurvedic Details
          </h2>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <NeuInput
              label="Full Name"
              type="text"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              required
            />

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
                Ayurvedic Prakriti (Constitution)
              </label>
              <select
                value={prakritiType}
                onChange={(e) => setPrakritiType(e.target.value)}
                className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
              >
                <option value="Vata">Vata Dominant</option>
                <option value="Pitta">Pitta Dominant</option>
                <option value="Kapha">Kapha Dominant</option>
                <option value="Vata-Pitta">Vata-Pitta</option>
                <option value="Pitta-Kapha">Pitta-Kapha</option>
                <option value="Vata-Kapha">Vata-Kapha</option>
                <option value="Tridoshic">Sama Dosha (Tridoshic)</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1.5">Blood Group</label>
              <select
                value={bloodGroup}
                onChange={(e) => setBloodGroup(e.target.value)}
                className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
              >
                <option value="A+">A+</option>
                <option value="A-">A-</option>
                <option value="B+">B+</option>
                <option value="B-">B-</option>
                <option value="AB+">AB+</option>
                <option value="AB-">AB-</option>
                <option value="O+">O+</option>
                <option value="O-">O-</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1.5">Registered Email</label>
              <input
                type="text"
                disabled
                value={user?.email || ''}
                placeholder="No registered email"
                className="w-full px-4 py-2.5 rounded-xl bg-[#EBE8DC] text-xs font-medium text-[#6F7B71] cursor-not-allowed"
              />
            </div>
          </div>
        </NeuCard>

        {/* Clinical History & Allergies */}
        <NeuCard elevation="raised" className="p-6 space-y-4">
          <h2 className="font-serif text-lg font-bold text-[#234737] border-b border-[#E3DFC9] pb-3">
            Known Allergies & Clinical History
          </h2>

          <div>
            <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
              Known Allergies (Comma separated oils, herbs, foods):
            </label>
            <input
              type="text"
              value={allergiesInput}
              onChange={(e) => setAllergiesInput(e.target.value)}
              placeholder="e.g. Sesame, Mustard oil, Shellfish..."
              className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none"
            />
          </div>

          <div>
            <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
              Medical History & Health Concerns:
            </label>
            <textarea
              value={medicalHistory}
              onChange={(e) => setMedicalHistory(e.target.value)}
              rows={3}
              placeholder="Detail any previous surgeries, spinal conditions, or chronic ailments..."
              className="w-full p-3 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] focus:outline-none"
            />
          </div>
        </NeuCard>

        {/* Emergency Contact */}
        <NeuCard elevation="raised" className="p-6 space-y-4">
          <h2 className="font-serif text-lg font-bold text-[#234737] border-b border-[#E3DFC9] pb-3">
            Emergency Contact Information
          </h2>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <NeuInput
              label="Contact Person Name"
              type="text"
              value={emergencyName}
              onChange={(e) => setEmergencyName(e.target.value)}
              leftIcon={<User className="w-4 h-4" />}
            />
            <NeuInput
              label="Contact Mobile Number"
              type="tel"
              value={emergencyPhone}
              onChange={(e) => setEmergencyPhone(e.target.value)}
              leftIcon={<Phone className="w-4 h-4" />}
            />
          </div>
        </NeuCard>

        <div className="flex justify-end">
          <NeuButton type="submit" variant="primary" size="lg" className="flex items-center gap-2">
            <Save className="w-4 h-4" /> Save Profile Changes
          </NeuButton>
        </div>
      </form>
    </div>
  );
};
