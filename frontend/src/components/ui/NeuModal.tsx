import React, { useEffect } from 'react';
import { X } from 'lucide-react';
import { NeuButton } from './NeuButton';

export interface NeuModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: React.ReactNode;
  description?: string;
  children: React.ReactNode;
  maxWidth?: 'sm' | 'md' | 'lg' | 'xl' | '2xl';
  showCloseButton?: boolean;
}

export const NeuModal: React.FC<NeuModalProps> = ({
  isOpen,
  onClose,
  title,
  description,
  children,
  maxWidth = 'md',
  showCloseButton = true,
}) => {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    if (isOpen) {
      document.body.style.overflow = 'hidden';
      window.addEventListener('keydown', handleKeyDown);
    }
    return () => {
      document.body.style.overflow = 'unset';
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const maxWidthClasses = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-2xl',
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6 overflow-y-auto">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-charcoal/40 backdrop-blur-sm transition-opacity"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Modal Surface */}
      <div
        role="dialog"
        aria-modal="true"
        className={`relative w-full ${maxWidthClasses[maxWidth]} bg-neu-bg rounded-neu-lg p-6 sm:p-8 border border-neu-border shadow-neu-raised-lg z-10 animate-in fade-in zoom-in-95 duration-200`}
      >
        <div className="flex items-start justify-between gap-4 mb-4">
          <div>
            {title && (
              <h2 className="text-xl font-bold text-charcoal tracking-tight">
                {title}
              </h2>
            )}
            {description && (
              <p className="text-sm text-slateneu mt-1">{description}</p>
            )}
          </div>
          {showCloseButton && (
            <NeuButton
              variant="flat"
              size="icon"
              onClick={onClose}
              aria-label="Close dialog"
              className="text-slateneu hover:text-charcoal shrink-0"
            >
              <X className="w-4 h-4" />
            </NeuButton>
          )}
        </div>

        <div>{children}</div>
      </div>
    </div>
  );
};
