import React from 'react';

export interface NeuCardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'raised' | 'raised-sm' | 'raised-lg' | 'inset' | 'flat';
  interactive?: boolean;
}

export const NeuCard: React.FC<NeuCardProps> = ({
  children,
  variant = 'raised',
  interactive = false,
  className = '',
  ...props
}) => {
  const variantClasses = {
    raised: 'neu-card',
    'raised-sm':
      'bg-neu-bg rounded-neu-sm border border-neu-border/60 shadow-neu-raised-sm',
    'raised-lg':
      'bg-neu-bg rounded-neu-lg border border-neu-border/70 shadow-neu-raised-lg',
    inset: 'neu-container-inset border border-neu-border/40',
    flat: 'bg-neu-bg rounded-neu border border-neu-border/60',
  };

  const interactiveClasses = interactive
    ? 'transition-all duration-200 hover:-translate-y-1 hover:shadow-neu-raised-lg cursor-pointer'
    : '';

  return (
    <div
      className={`p-6 ${variantClasses[variant]} ${interactiveClasses} ${className}`}
      {...props}
    >
      {children}
    </div>
  );
};
