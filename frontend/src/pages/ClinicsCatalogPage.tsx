import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Navbar } from '@/components/layout/Navbar';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { clinicApi } from '@/lib/api/client';
import { Clinic } from '@/lib/api/types';
import {
  Search,
  MapPin,
  Star,
  Building2,
  Calendar,
  Languages,
  Clock,
  ArrowRight,
  Filter,
} from 'lucide-react';

export const ClinicsCatalogPage: React.FC = () => {
  const [clinics, setClinics] = useState<Clinic[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedLanguage, setSelectedLanguage] = useState<string>('all');
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    clinicApi
      .list()
      .then((data) => setClinics(data))
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  }, []);

  const filteredClinics = clinics.filter((c) => {
    const matchesSearch =
      c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      c.address.toLowerCase().includes(searchQuery.toLowerCase()) ||
      c.description?.toLowerCase().includes(searchQuery.toLowerCase());

    const matchesLang =
      selectedLanguage === 'all' ||
      c.languages?.some((l) => l.toLowerCase().includes(selectedLanguage.toLowerCase()));

    return matchesSearch && matchesLang;
  });

  return (
    <div className="min-h-screen bg-[#F5F3EC] flex flex-col text-[#27352D]">
      <Navbar />

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-10">
        {/* Header */}
        <div className="mb-8">
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            Certified Ayurveda Centers
          </p>
          <h1 className="font-serif text-3xl sm:text-4xl font-bold text-[#234737]">
            Find Accredited Panchakarma Clinics
          </h1>
          <p className="text-sm text-[#6F7B71] mt-1 max-w-2xl">
            Choose from accredited centers equipped with specialized treatment rooms, licensed Ayurvedic doctors, and standardized therapy protocols.
          </p>
        </div>

        {/* Filter & Search Bar */}
        <NeuCard elevation="raised" className="p-4 mb-8">
          <div className="flex flex-col sm:flex-row items-center gap-4">
            <div className="relative flex-1 w-full">
              <Search className="w-4 h-4 text-[#6F7B71] absolute left-3.5 top-3" />
              <input
                type="text"
                placeholder="Search clinics by name, city (e.g. Kovalam, Bengaluru, Rishikesh)..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs text-[#27352D] placeholder-[#6F7B71]/70 focus:outline-none focus:ring-2 focus:ring-[#315C49]/40"
              />
            </div>

            <div className="flex items-center gap-2 w-full sm:w-auto">
              <Filter className="w-4 h-4 text-[#6F7B71]" />
              <select
                value={selectedLanguage}
                onChange={(e) => setSelectedLanguage(e.target.value)}
                className="px-3 py-2 rounded-xl bg-[#F5F3EC] neu-raised-sm text-xs text-[#27352D] font-medium focus:outline-none"
              >
                <option value="all">All Languages</option>
                <option value="Malayalam">Malayalam</option>
                <option value="Kannada">Kannada</option>
                <option value="Hindi">Hindi</option>
                <option value="English">English</option>
              </select>
            </div>
          </div>
        </NeuCard>

        {/* Clinics Grid */}
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <NeuFeedbackStates.Skeleton count={3} />
          </div>
        ) : filteredClinics.length === 0 ? (
          <NeuFeedbackStates.EmptyState
            title="No Clinics Found"
            description="No clinics match your current search criteria. Try removing filters or searching for another city."
            actionText="Clear Search"
            onAction={() => {
              setSearchQuery('');
              setSelectedLanguage('all');
            }}
          />
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {filteredClinics.map((clinic) => (
              <NeuCard
                key={clinic.id}
                elevation="raised"
                className="p-6 flex flex-col justify-between group hover:neu-raised-lg transition-all"
              >
                <div>
                  <div className="flex items-start justify-between gap-2 mb-3">
                    <span className="text-[11px] font-bold px-2.5 py-1 rounded-full bg-[#315C49]/10 text-[#315C49]">
                      {clinic.code}
                    </span>
                    <div className="flex items-center gap-1 text-xs font-bold text-[#B7792B]">
                      <Star className="w-3.5 h-3.5 fill-current" />
                      <span>{clinic.rating || 4.9}</span>
                    </div>
                  </div>

                  <h2 className="font-serif text-xl font-bold text-[#234737] group-hover:text-[#315C49] transition-colors mb-1">
                    {clinic.name}
                  </h2>

                  <p className="text-xs text-[#6F7B71] flex items-center gap-1.5 mb-3">
                    <MapPin className="w-3.5 h-3.5 text-[#C98769] shrink-0" />
                    <span>{clinic.address}</span>
                  </p>

                  <p className="text-xs text-[#27352D]/80 line-clamp-3 mb-4 leading-relaxed">
                    {clinic.description}
                  </p>

                  <div className="space-y-1.5 text-[11px] text-[#6F7B71] border-t border-[#E3DFC9] pt-3">
                    <div className="flex items-center gap-2">
                      <Clock className="w-3 h-3 text-[#315C49]" />
                      <span>Hours: {clinic.hours || '08:00 - 18:00 IST'}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Languages className="w-3 h-3 text-[#C98769]" />
                      <span>{clinic.languages?.join(', ') || 'English, Malayalam'}</span>
                    </div>
                  </div>
                </div>

                <div className="pt-5 mt-4 border-t border-[#E3DFC9] flex items-center gap-3">
                  <Link to={`/clinics/${clinic.id}`} className="flex-1">
                    <NeuButton variant="raised" size="sm" className="w-full">
                      View Details
                    </NeuButton>
                  </Link>
                  <Link to={`/patient/book?clinicId=${clinic.id}`} className="flex-1">
                    <NeuButton variant="primary" size="sm" className="w-full flex items-center justify-center gap-1">
                      Book Now <ArrowRight className="w-3 h-3" />
                    </NeuButton>
                  </Link>
                </div>
              </NeuCard>
            ))}
          </div>
        )}
      </main>
    </div>
  );
};
