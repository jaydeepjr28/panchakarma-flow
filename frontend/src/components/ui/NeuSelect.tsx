import React, { forwardRef } from 'react';
import { ChevronDown } from 'lucide-react';

export interface NeuSelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface NeuSelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  options: NeuSelectOption[];
  error?: string;
  helperText?: string;
}

export const NeuSelect = forwardRef<HTMLSelectElement, NeuSelectProps>(
  ({ label, options, error, helperText, className = '', id, ...props }, ref) => {
    const selectId = id || (label ? label.toLowerCase().replace(/\s+/g, '-') : undefined);

    return (
      <div className="w-full space-y-1.5">
        {label && (
          <label
            htmlFor={selectId}
            className="block text-xs font-semibold uppercase tracking-wider text-charcoal/80"
          >
            {label}
          </label>
        )}
        <div className="relative">
          <select
            id={selectId}
            ref={ref}
            className={`w-full py-2.5 px-4 pr-10 rounded-neu-sm text-sm neu-input appearance-none bg-neu-bg cursor-pointer ${
              error ? 'border-neuError ring-1 ring-neuError/40' : ''
            } ${className}`}
            {...props}
          >
            {options.map((opt) => (
              <option key={opt.value} value={opt.value} disabled={opt.disabled}>
                {opt.label}
              </option>
            ))}
          </select>
          <div className="absolute right-3.5 top-1/2 -translate-y-1/2 pointer-events-none text-slateneu">
            <ChevronDown className="w-4 h-4" />
          </div>
        </div>
        {error ? (
          <p className="text-xs font-medium text-neuError mt-1">{error}</p>
        ) : helperText ? (
          <p className="text-xs text-slateneu mt-1">{helperText}</p>
        ) : null}
      </div>
    );
  }
);

NeuSelect.displayName = 'NeuSelect';
