import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Navbar } from '@/components/layout/Navbar';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { clinicApi } from '@/lib/api/client';
import { Clinic } from '@/lib/api/types';
import {
  Sparkles,
  Calendar,
  Clock,
  ShieldCheck,
  Activity,
  HeartHandshake,
  CheckCircle2,
  ArrowRight,
  Compass,
  FileCheck2,
  Stethoscope,
  Building2,
  Star,
  MapPin,
  ChevronRight,
} from 'lucide-react';

export const LandingPage: React.FC = () => {
  const [featuredClinics, setFeaturedClinics] = useState<Clinic[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    clinicApi
      .list()
      .then((data) => setFeaturedClinics(data.slice(0, 3)))
      .catch(() => {})
      .finally(() => setIsLoading(false));
  }, []);

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col text-[#27352D]">
      <Navbar />

      {/* Hero Section */}
      <section className="relative overflow-hidden py-16 sm:py-24 px-4 sm:px-6 lg:px-8 border-b border-[#E3DFC9]/60">
        <div className="max-w-7xl mx-auto">
          <div className="text-center max-w-3xl mx-auto space-y-6">
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-2xl bg-[#F5F3EC] neu-raised-sm border border-[#E3DFC9]">
              <Sparkles className="w-4 h-4 text-[#C98769]" />
              <span className="text-xs font-semibold uppercase tracking-wider text-[#315C49]">
                Smart India Hackathon • Ayush Protocol Innovation
              </span>
            </div>

            <h1 className="font-serif text-4xl sm:text-5xl lg:text-6xl font-bold text-[#234737] tracking-tight leading-tight">
              Classical Panchakarma,{' '}
              <span className="text-[#315C49] underline decoration-[#C98769] decoration-wavy decoration-2">
                Digitally Mastered.
              </span>
            </h1>

            <p className="text-base sm:text-lg text-[#6F7B71] leading-relaxed">
              An enterprise healthcare platform bridging traditional Ayurvedic Shodhana therapy with modern clinical rigor — from 10-minute hold bookings and pre-care checklists to real-time step tracking and complication safeguards.
            </p>

            {/* Call to Actions */}
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4 pt-4">
              <Link to="/clinics">
                <NeuButton variant="primary" size="lg" className="w-full sm:w-auto flex items-center justify-center gap-2">
                  <Compass className="w-5 h-5" />
                  Explore Clinics & Packages
                  <ArrowRight className="w-4 h-4 ml-1" />
                </NeuButton>
              </Link>
              <Link to="/patient/book">
                <NeuButton variant="raised" size="lg" className="w-full sm:w-auto flex items-center justify-center gap-2">
                  <Calendar className="w-5 h-5 text-[#C98769]" />
                  Book Appointment Now
                </NeuButton>
              </Link>
            </div>

            {/* Trust Badges */}
            <div className="pt-8 flex flex-wrap items-center justify-center gap-6 text-xs text-[#6F7B71]">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-[#357A55]" />
                <span>Microservices Architecture</span>
              </div>
              <div className="flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 text-[#357A55]" />
                <span>NABH & Ayush Protocol Standard</span>
              </div>
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-[#357A55]" />
                <span>Provisional Hold Conflict Resolution</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 4-Stage Workflow Section */}
      <section className="py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto w-full">
        <div className="text-center max-w-2xl mx-auto mb-12">
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            End-to-End Care Cycle
          </p>
          <h2 className="font-serif text-3xl font-bold text-[#234737]">
            How PanchakarmaFlow Works
          </h2>
          <p className="text-sm text-[#6F7B71] mt-2">
            A continuous, verifiable journey ensuring patient safety and treatment efficacy.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          {/* Step 1 */}
          <NeuCard elevation="raised" className="p-6 relative group hover:neu-raised-lg transition-all">
            <div className="w-12 h-12 rounded-2xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center font-bold text-lg mb-4 neu-inset">
              01
            </div>
            <h3 className="font-bold text-lg text-[#234737] mb-2">Discover</h3>
            <p className="text-xs text-[#6F7B71] leading-relaxed">
              Explore accredited clinics, certified Vaidyas, and tailored therapy packages like Vamana, Virechana, or Shirodhara.
            </p>
            <div className="mt-4 pt-3 border-t border-[#E3DFC9] text-[11px] font-semibold text-[#315C49] flex items-center gap-1">
              Verified package protocols <ChevronRight className="w-3.5 h-3.5" />
            </div>
          </NeuCard>

          {/* Step 2 */}
          <NeuCard elevation="raised" className="p-6 relative group hover:neu-raised-lg transition-all">
            <div className="w-12 h-12 rounded-2xl bg-[#C98769]/15 text-[#C98769] flex items-center justify-center font-bold text-lg mb-4 neu-inset">
              02
            </div>
            <h3 className="font-bold text-lg text-[#234737] mb-2">Book with Hold</h3>
            <p className="text-xs text-[#6F7B71] leading-relaxed">
              Select doctor and compatible room with a real 10-minute hold token, eliminating double-bookings and race conditions.
            </p>
            <div className="mt-4 pt-3 border-t border-[#E3DFC9] text-[11px] font-semibold text-[#C98769] flex items-center gap-1">
              Deterministic slot locks <ChevronRight className="w-3.5 h-3.5" />
            </div>
          </NeuCard>

          {/* Step 3 */}
          <NeuCard elevation="raised" className="p-6 relative group hover:neu-raised-lg transition-all">
            <div className="w-12 h-12 rounded-2xl bg-[#A8B9A0]/30 text-[#234737] flex items-center justify-center font-bold text-lg mb-4 neu-inset">
              03
            </div>
            <h3 className="font-bold text-lg text-[#234737] mb-2">Prepare</h3>
            <p className="text-xs text-[#6F7B71] leading-relaxed">
              Complete mandatory pre-procedure fasting, hydration, and lifestyle checklists reviewed by attending staff prior to check-in.
            </p>
            <div className="mt-4 pt-3 border-t border-[#E3DFC9] text-[11px] font-semibold text-[#234737] flex items-center gap-1">
              Verified patient readiness <ChevronRight className="w-3.5 h-3.5" />
            </div>
          </NeuCard>

          {/* Step 4 */}
          <NeuCard elevation="raised" className="p-6 relative group hover:neu-raised-lg transition-all">
            <div className="w-12 h-12 rounded-2xl bg-[#357A55]/15 text-[#357A55] flex items-center justify-center font-bold text-lg mb-4 neu-inset">
              04
            </div>
            <h3 className="font-bold text-lg text-[#234737] mb-2">Track & Protect</h3>
            <p className="text-xs text-[#6F7B71] leading-relaxed">
              Therapists log ordered step sequences with server timestamps, consumable batches, and instant doctor complication escalation.
            </p>
            <div className="mt-4 pt-3 border-t border-[#E3DFC9] text-[11px] font-semibold text-[#357A55] flex items-center gap-1">
              Real-time audit trail <ChevronRight className="w-3.5 h-3.5" />
            </div>
          </NeuCard>
        </div>
      </section>

      {/* Featured Clinics Showcase */}
      <section className="py-16 bg-[#EFECE2]/50 border-t border-b border-[#E3DFC9]/70 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between mb-8 gap-4">
            <div>
              <p className="text-xs uppercase tracking-widest font-bold text-[#315C49] mb-1">
                Accredited Centers
              </p>
              <h2 className="font-serif text-3xl font-bold text-[#234737]">
                Featured Ayurvedic Clinics
              </h2>
            </div>
            <Link to="/clinics">
              <NeuButton variant="raised" size="sm" className="flex items-center gap-2">
                View All Clinics <ArrowRight className="w-4 h-4" />
              </NeuButton>
            </Link>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {featuredClinics.map((clinic) => (
              <NeuCard key={clinic.id} elevation="raised" className="p-6 flex flex-col justify-between">
                <div>
                  <div className="flex items-start justify-between gap-2 mb-3">
                    <span className="text-xs font-semibold px-2.5 py-1 rounded-full bg-[#315C49]/10 text-[#315C49]">
                      {clinic.code}
                    </span>
                    <div className="flex items-center gap-1 text-xs font-bold text-[#B7792B]">
                      <Star className="w-3.5 h-3.5 fill-current" />
                      <span>{clinic.rating || 4.9}</span>
                    </div>
                  </div>

                  <h3 className="font-serif text-xl font-bold text-[#234737] mb-1">{clinic.name}</h3>
                  <p className="text-xs text-[#6F7B71] flex items-center gap-1.5 mb-3">
                    <MapPin className="w-3.5 h-3.5 text-[#C98769]" />
                    {clinic.address}
                  </p>
                  <p className="text-xs text-[#27352D]/80 line-clamp-2 mb-4">{clinic.description}</p>
                </div>

                <div className="pt-4 border-t border-[#E3DFC9] flex items-center justify-between">
                  <div className="text-[11px] text-[#6F7B71]">
                    <span>Languages: </span>
                    <span className="font-medium text-[#27352D]">{clinic.languages?.join(', ') || 'English, Hindi'}</span>
                  </div>
                  <Link to={`/clinics/${clinic.id}`}>
                    <NeuButton variant="primary" size="sm">
                      Details
                    </NeuButton>
                  </Link>
                </div>
              </NeuCard>
            ))}
          </div>
        </div>
      </section>

      {/* Role Personas Quick Entry */}
      <section className="py-16 px-4 sm:px-6 lg:px-8 max-w-7xl mx-auto w-full">
        <div className="text-center max-w-2xl mx-auto mb-10">
          <h2 className="font-serif text-3xl font-bold text-[#234737]">
            Tailored Experiences by Role
          </h2>
          <p className="text-sm text-[#6F7B71] mt-2">
            Each stakeholder enjoys a specialized interface tailored to their exact workflow.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <NeuCard elevation="raised" className="p-6 text-center">
            <div className="w-14 h-14 mx-auto rounded-2xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center mb-4 neu-inset">
              <HeartHandshake className="w-7 h-7" />
            </div>
            <h3 className="font-bold text-lg text-[#234737] mb-2">Patients</h3>
            <p className="text-xs text-[#6F7B71] mb-6">
              Book therapies with live availability, complete pre-care checklists, track step progress, and view personalized post-care instructions.
            </p>
            <Link to="/patient/dashboard">
              <NeuButton variant="raised" size="sm" className="w-full">
                Enter Patient Portal
              </NeuButton>
            </Link>
          </NeuCard>

          <NeuCard elevation="raised" className="p-6 text-center">
            <div className="w-14 h-14 mx-auto rounded-2xl bg-[#C98769]/15 text-[#C98769] flex items-center justify-center mb-4 neu-inset">
              <Building2 className="w-7 h-7" />
            </div>
            <h3 className="font-bold text-lg text-[#234737] mb-2">Clinic Administrators</h3>
            <p className="text-xs text-[#6F7B71] mb-6">
              Manage doctor schedules, leaves, room assignments, therapy packages, and monitor daily operational and financial metrics.
            </p>
            <Link to="/clinic/dashboard">
              <NeuButton variant="raised" size="sm" className="w-full">
                Enter Admin Dashboard
              </NeuButton>
            </Link>
          </NeuCard>

          <NeuCard elevation="raised" className="p-6 text-center">
            <div className="w-14 h-14 mx-auto rounded-2xl bg-[#357A55]/15 text-[#357A55] flex items-center justify-center mb-4 neu-inset">
              <Activity className="w-7 h-7" />
            </div>
            <h3 className="font-bold text-lg text-[#234737] mb-2">Therapists & Vaidyas</h3>
            <p className="text-xs text-[#6F7B71] mb-6">
              View assigned queue, verify patient vitals, execute ordered steps with countdown timers, log materials, and escalate complications.
            </p>
            <Link to="/therapist/dashboard">
              <NeuButton variant="raised" size="sm" className="w-full">
                Enter Therapist Workspace
              </NeuButton>
            </Link>
          </NeuCard>
        </div>
      </section>

      {/* Footer */}
      <footer className="mt-auto bg-[#EBE8DC] border-t border-[#E3DFC9] py-12 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-6">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-[#315C49] text-white flex items-center justify-center font-bold">
              PF
            </div>
            <div>
              <p className="font-serif font-bold text-sm text-[#234737]">PanchakarmaFlow Protocol</p>
              <p className="text-[11px] text-[#6F7B71]">Smart India Hackathon • AYUSH Healthcare Track</p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-6 text-xs text-[#6F7B71]">
            <Link to="/clinics" className="hover:text-[#234737]">Clinics</Link>
            <Link to="/patient/book" className="hover:text-[#234737]">Book Therapy</Link>
            <Link to="/login" className="hover:text-[#234737]">Sign In</Link>
            <Link to="/register" className="hover:text-[#234737]">Register Patient</Link>
          </div>

          <div className="text-[11px] text-[#6F7B71]">
            © 2026 PanchakarmaFlow. Built with Go microservices & Neumorphic Design.
          </div>
        </div>
      </footer>
    </div>
  );
};
