import React from 'react';

export interface NeuBadgeProps {
  children: React.ReactNode;
  variant?:
    | 'forest'
    | 'terracotta'
    | 'sage'
    | 'neutral'
    | 'success'
    | 'warning'
    | 'error'
    | 'outline';
  size?: 'sm' | 'md';
  className?: string;
  dot?: boolean;
}

export const NeuBadge: React.FC<NeuBadgeProps> = ({
  children,
  variant = 'forest',
  size = 'md',
  className = '',
  dot = false,
}) => {
  const sizeClasses = {
    sm: 'text-[11px] px-2.5 py-0.5 rounded-full font-medium',
    md: 'text-xs px-3 py-1 rounded-full font-semibold',
  };

  const variantClasses = {
    forest: 'bg-forest/10 text-forest border border-forest/20 shadow-neu-raised-xs',
    terracotta:
      'bg-terracotta/15 text-terracotta-dark border border-terracotta/30 shadow-neu-raised-xs',
    sage: 'bg-sage/25 text-charcoal border border-sage/40 shadow-neu-raised-xs',
    neutral: 'bg-white/60 text-charcoal border border-neu-border shadow-neu-raised-xs',
    success: 'bg-emerald-50 text-emerald-800 border border-emerald-200 shadow-neu-raised-xs',
    warning: 'bg-amber-50 text-amber-800 border border-amber-200 shadow-neu-raised-xs',
    error: 'bg-rose-50 text-rose-800 border border-rose-200 shadow-neu-raised-xs',
    outline: 'bg-transparent text-charcoal border border-neu-border',
  };

  const dotColors = {
    forest: 'bg-forest',
    terracotta: 'bg-terracotta',
    sage: 'bg-sage-dark',
    neutral: 'bg-slateneu',
    success: 'bg-emerald-600',
    warning: 'bg-amber-500',
    error: 'bg-rose-600',
    outline: 'bg-charcoal',
  };

  return (
    <span
      className={`inline-flex items-center gap-1.5 ${sizeClasses[size]} ${variantClasses[variant]} ${className}`}
    >
      {dot && <span className={`w-1.5 h-1.5 rounded-full ${dotColors[variant]}`} />}
      {children}
    </span>
  );
};
