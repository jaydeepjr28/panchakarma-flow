import React from 'react';
import { NeuCard } from './NeuCard';
import { ArrowUpRight, ArrowDownRight } from 'lucide-react';

export interface NeuStatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: React.ReactNode;
  trend?: {
    value: string;
    isPositive?: boolean;
    label?: string;
  };
  accentColor?: 'forest' | 'terracotta' | 'sage' | 'charcoal';
  className?: string;
}

export const NeuStatCard: React.FC<NeuStatCardProps> = ({
  title,
  value,
  subtitle,
  icon,
  trend,
  accentColor = 'forest',
  className = '',
}) => {
  const accentClasses = {
    forest: 'text-forest bg-forest/10 border-forest/20',
    terracotta: 'text-terracotta bg-terracotta/15 border-terracotta/30',
    sage: 'text-charcoal bg-sage/20 border-sage/40',
    charcoal: 'text-charcoal bg-charcoal/10 border-charcoal/20',
  };

  return (
    <NeuCard variant="raised-sm" className={`p-5 relative overflow-hidden ${className}`}>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-slateneu mb-1">
            {title}
          </p>
          <div className="text-2xl sm:text-3xl font-extrabold text-charcoal tracking-tight">
            {value}
          </div>
          {subtitle && (
            <p className="text-xs text-slateneu mt-1">{subtitle}</p>
          )}
        </div>
        {icon && (
          <div
            className={`w-11 h-11 rounded-neu-sm flex items-center justify-center border shadow-neu-raised-xs ${accentClasses[accentColor]}`}
          >
            {icon}
          </div>
        )}
      </div>

      {trend && (
        <div className="flex items-center gap-1.5 mt-3 pt-3 border-t border-neu-border/50 text-xs font-medium">
          <span
            className={`inline-flex items-center gap-0.5 font-bold ${
              trend.isPositive ? 'text-neuSuccess' : 'text-neuError'
            }`}
          >
            {trend.isPositive ? (
              <ArrowUpRight className="w-3.5 h-3.5" />
            ) : (
              <ArrowDownRight className="w-3.5 h-3.5" />
            )}
            {trend.value}
          </span>
          {trend.label && <span className="text-slateneu">{trend.label}</span>}
        </div>
      )}
    </NeuCard>
  );
};
