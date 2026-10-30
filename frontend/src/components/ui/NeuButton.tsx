import React from 'react';
import { Loader2 } from 'lucide-react';

export interface NeuButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'default' | 'primary' | 'accent' | 'flat' | 'ghost' | 'danger';
  size?: 'sm' | 'md' | 'lg' | 'icon';
  isLoading?: boolean;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
}

export const NeuButton: React.FC<NeuButtonProps> = ({
  children,
  variant = 'default',
  size = 'md',
  isLoading = false,
  leftIcon,
  rightIcon,
  className = '',
  disabled,
  ...props
}) => {
  const baseClasses =
    'inline-flex items-center justify-center font-medium transition-all duration-200 select-none cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-forest focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-60';

  const sizeClasses = {
    sm: 'text-xs px-3.5 py-1.5 rounded-neu-sm gap-1.5',
    md: 'text-sm px-5 py-2.5 rounded-neu gap-2',
    lg: 'text-base px-6 py-3 rounded-neu-lg gap-2.5',
    icon: 'p-2.5 rounded-neu aspect-square',
  };

  const variantClasses = {
    default: 'neu-button text-charcoal border border-neu-border/60 hover:text-forest',
    primary: 'neu-button-primary font-semibold',
    accent: 'neu-button-accent font-semibold',
    danger:
      'bg-neuError text-white shadow-neu-raised-sm hover:brightness-105 active:shadow-neu-pressed',
    flat: 'bg-neu-bg text-charcoal hover:bg-[#ECE8DF] active:bg-[#E3DFD4]',
    ghost: 'bg-transparent text-slateneu hover:text-charcoal hover:bg-black/5 active:bg-black/10',
  };

  return (
    <button
      className={`${baseClasses} ${sizeClasses[size]} ${variantClasses[variant]} ${className}`}
      disabled={disabled || isLoading}
      {...props}
    >
      {isLoading ? (
        <Loader2 className="w-4 h-4 animate-spin text-current" />
      ) : (
        <>
          {leftIcon && <span className="inline-flex shrink-0">{leftIcon}</span>}
          {children}
          {rightIcon && <span className="inline-flex shrink-0">{rightIcon}</span>}
        </>
      )}
    </button>
  );
};
