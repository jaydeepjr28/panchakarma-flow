import React from 'react';
import { NeuCard } from './NeuCard';
import { NeuButton } from './NeuButton';
import { AlertCircle, Inbox, RefreshCw } from 'lucide-react';

export interface NeuSkeletonProps {
  className?: string;
  count?: number;
}

export const NeuSkeleton: React.FC<NeuSkeletonProps> = ({ className = '', count = 1 }) => {
  return (
    <>
      {Array.from({ length: count }).map((_, idx) => (
        <div
          key={idx}
          className={`h-28 animate-pulse bg-[#E7E3D8] neu-inset rounded-2xl ${className}`}
          aria-hidden="true"
        />
      ))}
    </>
  );
};

export interface NeuEmptyStateProps {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  actionText?: string;
  onAction?: () => void;
  className?: string;
}

export const NeuEmptyState: React.FC<NeuEmptyStateProps> = ({
  title,
  description,
  icon,
  actionText,
  onAction,
  className = '',
}) => {
  return (
    <NeuCard elevation="flat" className={`text-center py-12 px-6 ${className}`}>
      <div className="w-14 h-14 mx-auto mb-4 rounded-2xl flex items-center justify-center bg-[#F5F3EC] border border-[#E3DFC9] neu-raised-sm text-[#6F7B71]">
        {icon || <Inbox className="w-7 h-7 text-[#315C49]/70" />}
      </div>
      <h3 className="text-lg font-bold text-[#234737]">{title}</h3>
      {description && (
        <p className="text-xs text-[#6F7B71] mt-1.5 max-w-md mx-auto">
          {description}
        </p>
      )}
      {actionText && onAction && (
        <div className="mt-5">
          <NeuButton variant="primary" size="sm" onClick={onAction}>
            {actionText}
          </NeuButton>
        </div>
      )}
    </NeuCard>
  );
};

export interface NeuErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  className?: string;
}

export const NeuErrorState: React.FC<NeuErrorStateProps> = ({
  title = 'Something went wrong',
  message,
  onRetry,
  className = '',
}) => {
  return (
    <NeuCard
      elevation="raised-sm"
      className={`border-l-4 border-l-[#B74747] p-6 ${className}`}
    >
      <div className="flex items-start gap-3.5">
        <div className="p-2 rounded-xl bg-rose-50 text-[#B74747] shrink-0">
          <AlertCircle className="w-5 h-5" />
        </div>
        <div className="flex-1">
          <h4 className="text-sm font-bold text-[#234737]">{title}</h4>
          <p className="text-xs text-[#6F7B71] mt-1">{message}</p>
          {onRetry && (
            <NeuButton
              variant="raised"
              size="sm"
              onClick={onRetry}
              className="mt-3 text-xs"
            >
              Retry
            </NeuButton>
          )}
        </div>
      </div>
    </NeuCard>
  );
};

export const NeuFeedbackStates = {
  Skeleton: NeuSkeleton,
  EmptyState: NeuEmptyState,
  ErrorState: NeuErrorState,
};
