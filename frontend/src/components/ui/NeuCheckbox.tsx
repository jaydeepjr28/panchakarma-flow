import React, { forwardRef } from 'react';
import { Check } from 'lucide-react';

export interface NeuCheckboxProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label: React.ReactNode;
  description?: string;
  error?: string;
}

export const NeuCheckbox = forwardRef<HTMLInputElement, NeuCheckboxProps>(
  ({ label, description, error, checked, className = '', id, onChange, ...props }, ref) => {
    const checkboxId = id || (typeof label === 'string' ? label.toLowerCase().replace(/\s+/g, '-') : undefined);

    return (
      <div className={`space-y-1 ${className}`}>
        <label
          htmlFor={checkboxId}
          className="flex items-start gap-3 cursor-pointer select-none group"
        >
          <div className="relative mt-0.5 shrink-0">
            <input
              id={checkboxId}
              ref={ref}
              type="checkbox"
              checked={checked}
              onChange={onChange}
              className="sr-only"
              {...props}
            />
            <div
              className={`w-5 h-5 rounded-md flex items-center justify-center transition-all duration-200 border ${
                checked
                  ? 'bg-forest text-white border-forest shadow-neu-raised-xs'
                  : 'bg-neu-bg border-neu-border neu-inset-sm group-hover:border-forest/60'
              } ${error ? 'border-neuError ring-1 ring-neuError' : ''}`}
            >
              {checked && <Check className="w-3.5 h-3.5 stroke-[3]" />}
            </div>
          </div>
          <div>
            <span className="text-sm font-medium text-charcoal leading-snug">
              {label}
            </span>
            {description && (
              <p className="text-xs text-slateneu mt-0.5 leading-relaxed">
                {description}
              </p>
            )}
          </div>
        </label>
        {error && <p className="text-xs font-medium text-neuError pl-8">{error}</p>}
      </div>
    );
  }
);

NeuCheckbox.displayName = 'NeuCheckbox';
