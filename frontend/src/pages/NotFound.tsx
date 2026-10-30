import React from 'react';
import { Link } from 'react-router-dom';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { Compass, ArrowLeft } from 'lucide-react';

export default function NotFoundPage() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-[#F5F3EC] p-6 text-center text-[#27352D]">
      <NeuCard elevation="raised-lg" className="p-8 max-w-md w-full space-y-6">
        <div className="w-16 h-16 mx-auto rounded-3xl bg-[#315C49]/10 text-[#315C49] flex items-center justify-center neu-inset">
          <Compass className="w-8 h-8 text-[#C98769]" />
        </div>

        <div className="space-y-2">
          <h1 className="font-serif text-5xl font-bold text-[#234737]">404</h1>
          <h2 className="font-serif text-xl font-bold text-[#27352D]">Pathway Not Found</h2>
          <p className="text-xs text-[#6F7B71] leading-relaxed">
            The treatment page or resource you requested could not be located in our clinical registry.
          </p>
        </div>

        <div className="flex flex-col sm:flex-row gap-3 justify-center pt-2">
          <Link to="/" className="w-full">
            <NeuButton variant="primary" size="md" className="w-full flex items-center justify-center gap-2">
              <ArrowLeft className="w-4 h-4" /> Return to Homepage
            </NeuButton>
          </Link>
          <NeuButton
            variant="raised"
            size="md"
            onClick={() => window.history.back()}
            className="w-full"
          >
            Go Back
          </NeuButton>
        </div>
      </NeuCard>
    </div>
  );
}
