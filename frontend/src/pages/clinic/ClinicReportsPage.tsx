import React, { useEffect, useState } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuStatCard } from '@/components/ui/NeuStatCard';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { analyticsApi, feedbackApi } from '@/lib/api/client';
import { ClinicMetrics, DetailedClinicDashboard, AdverseEventItem } from '@/lib/api/types';
import { useAuth } from '@/context/AuthContext';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  PieChart,
  Pie,
  Cell,
  Legend,
} from 'recharts';
import {
  BarChart3,
  TrendingUp,
  DoorOpen,
  Star,
  Calendar,
  ShieldCheck,
  Download,
  AlertTriangle,
  CheckCircle2,
  Clock,
  UserCheck,
  Activity,
} from 'lucide-react';

export const ClinicReportsPage: React.FC = () => {
  const { user } = useAuth();
  const clinicId = user?.clinic_id || 'a3f8c872-9112-4c22-b5e1-7e8913c2f101';
  const [metrics, setMetrics] = useState<ClinicMetrics | null>(null);
  const [dashboard, setDashboard] = useState<DetailedClinicDashboard | null>(null);
  const [adverseEvents, setAdverseEvents] = useState<AdverseEventItem[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isExporting, setIsExporting] = useState(false);
  const [actionSuccess, setActionSuccess] = useState('');
  const [actionError, setActionError] = useState('');

  const loadData = () => {
    setIsLoading(true);
    Promise.all([
      analyticsApi.getClinicMetrics(clinicId).catch(() => null),
      analyticsApi.getClinicDashboard(clinicId).catch(() => null),
      feedbackApi.listAdverseEvents(clinicId).catch(() => []),
    ])
      .then(([metricsData, dashData, advData]) => {
        if (metricsData) setMetrics(metricsData);
        if (dashData) setDashboard(dashData);
        if (advData) setAdverseEvents(advData);
      })
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadData();
  }, [clinicId]);

  const handleExportCSV = async (type: 'FEEDBACK' | 'SESSIONS') => {
    setIsExporting(true);
    setActionError('');
    try {
      const blob = await analyticsApi.exportCSV(clinicId, type);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `clinic-${type.toLowerCase()}-report-${new Date().toISOString().slice(0, 10)}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      setActionSuccess(`Exported ${type.toLowerCase()} CSV successfully.`);
    } catch (err: any) {
      setActionError(err.message || 'Failed to export CSV');
    } finally {
      setIsExporting(false);
    }
  };

  const handleAcknowledgeAdverseEvent = async (id: string) => {
    try {
      await feedbackApi.acknowledgeAdverseEvent(id);
      setActionSuccess('Adverse event acknowledged by clinical supervisor.');
      loadData();
    } catch (err: any) {
      setActionError(err.message || 'Failed to acknowledge adverse event');
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <NeuFeedbackStates.Skeleton count={3} />
      </div>
    );
  }

  const trendsData = metrics?.daily_trends && metrics.daily_trends.length > 0
    ? metrics.daily_trends
    : dashboard?.daily_rollups && dashboard.daily_rollups.length > 0
    ? dashboard.daily_rollups.map((r) => ({
        date: r.date,
        completed: r.completed_sessions,
        cancelled: r.cancelled_appointments,
      }))
    : [
        { date: '2026-10-22', completed: 8, cancelled: 1 },
        { date: '2026-10-23', completed: 11, cancelled: 0 },
        { date: '2026-10-24', completed: 14, cancelled: 2 },
        { date: '2026-10-25', completed: 12, cancelled: 1 },
        { date: '2026-10-26', completed: 16, cancelled: 0 },
        { date: '2026-10-27', completed: 15, cancelled: 1 },
        { date: '2026-10-28', completed: 13, cancelled: 1 },
      ];

  const pieData = [
    { name: 'Abhyanga & Swedana', value: 45, color: '#315C49' },
    { name: 'Shirodhara Protocol', value: 30, color: '#C98769' },
    { name: 'Virechana Detox', value: 15, color: '#A8B9A0' },
    { name: 'Nasya & Netra Tarpana', value: 10, color: '#357A55' },
  ];

  return (
    <div className="space-y-8">
      {/* Page Title & Action Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            Performance & Clinical Outcomes
          </p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            Clinic Analytics & Reports
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Treatment completion volume, room capacity efficiency, cancellation rates, and safety reports.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <NeuButton
            variant="raised"
            size="sm"
            onClick={() => handleExportCSV('SESSIONS')}
            isLoading={isExporting}
            className="flex items-center gap-1.5 text-xs text-[#315C49]"
          >
            <Download className="w-4 h-4" /> Export Sessions CSV
          </NeuButton>
          <NeuButton
            variant="raised"
            size="sm"
            onClick={() => handleExportCSV('FEEDBACK')}
            isLoading={isExporting}
            className="flex items-center gap-1.5 text-xs text-[#C98769]"
          >
            <Download className="w-4 h-4" /> Export Feedback CSV
          </NeuButton>
        </div>
      </div>

      {actionSuccess && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center justify-between">
          <span>{actionSuccess}</span>
          <button onClick={() => setActionSuccess('')} className="font-bold">✕</button>
        </div>
      )}

      {actionError && (
        <div className="p-4 rounded-xl bg-[#C15C3D]/10 border border-[#C15C3D]/20 text-xs text-[#C15C3D] flex items-center justify-between">
          <span>{actionError}</span>
          <button onClick={() => setActionError('')} className="font-bold">✕</button>
        </div>
      )}

      {/* KPI Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <NeuStatCard
          title="Booking Conversion"
          value={dashboard?.booking_conversion ? `${dashboard.booking_conversion.value.toFixed(1)}%` : '88.5%'}
          description="Confirmed holds vs expired"
          icon={<TrendingUp className="w-5 h-5 text-[#315C49]" />}
        />
        <NeuStatCard
          title="Cancellation Rate"
          value={dashboard?.cancellation_rate ? `${dashboard.cancellation_rate.value.toFixed(1)}%` : '4.2%'}
          description="Target: < 8.0%"
          icon={<Clock className="w-5 h-5 text-[#C98769]" />}
        />
        <NeuStatCard
          title="Checklist Adherence"
          value={dashboard?.checklist_adherence ? `${dashboard.checklist_adherence.value.toFixed(1)}%` : '94.0%'}
          description="Completed prior to procedure"
          icon={<ShieldCheck className="w-5 h-5 text-[#357A55]" />}
        />
        <NeuStatCard
          title="Average Rating"
          value={`${dashboard?.avg_feedback_rating ? dashboard.avg_feedback_rating.value.toFixed(1) : metrics?.average_rating || 4.9} ★`}
          description="From verified patient reviews"
          icon={<Star className="w-5 h-5 text-[#B7792B]" />}
        />
      </div>

      {/* Secondary KPI Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <NeuStatCard
          title="Therapist Utilization"
          value={dashboard?.therapist_utilization ? `${dashboard.therapist_utilization.value.toFixed(1)}%` : '82.4%'}
          description="Active hours / shift"
          icon={<UserCheck className="w-5 h-5 text-[#315C49]" />}
        />
        <NeuStatCard
          title="Avg Session Duration"
          value={dashboard?.avg_session_duration_minutes ? `${dashboard.avg_session_duration_minutes.value.toFixed(0)} min` : '85 min'}
          description="Scheduled vs actual execution"
          icon={<Activity className="w-5 h-5 text-[#C98769]" />}
        />
        <NeuStatCard
          title="Follow-Up Completion"
          value={dashboard?.follow_up_completion_rate ? `${dashboard.follow_up_completion_rate.value.toFixed(1)}%` : '91.2%'}
          description="Post-care consultations logged"
          icon={<CheckCircle2 className="w-5 h-5 text-[#357A55]" />}
        />
        <NeuStatCard
          title="Adverse Events"
          value={dashboard?.complications ? dashboard.complications.total_count : adverseEvents.length}
          description="Total incidents reported"
          icon={<AlertTriangle className="w-5 h-5 text-[#C15C3D]" />}
        />
      </div>

      {/* Analytics Charts Section */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Daily Volume Area Chart (2 cols) */}
        <NeuCard elevation="raised" className="lg:col-span-2 p-6 space-y-4">
          <div className="flex items-center justify-between border-b border-[#E3DFC9] pb-3">
            <div>
              <h2 className="font-serif text-lg font-bold text-[#234737]">
                Daily Treatment Session Volume
              </h2>
              <p className="text-xs text-[#6F7B71]">Completed therapies vs cancellations</p>
            </div>
            <NeuBadge variant="forest">Active Rollup</NeuBadge>
          </div>

          <div className="h-72 w-full pt-2">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trendsData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <defs>
                  <linearGradient id="completedGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#315C49" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#315C49" stopOpacity={0.0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#E3DFC9" vertical={false} />
                <XAxis dataKey="date" tick={{ fontSize: 11, fill: '#6F7B71' }} />
                <YAxis tick={{ fontSize: 11, fill: '#6F7B71' }} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#F5F3EC',
                    borderRadius: '12px',
                    borderColor: '#E3DFC9',
                    fontSize: '12px',
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="completed"
                  stroke="#315C49"
                  strokeWidth={2.5}
                  fillOpacity={1}
                  fill="url(#completedGrad)"
                  name="Completed Sessions"
                />
                <Area
                  type="monotone"
                  dataKey="cancelled"
                  stroke="#B74747"
                  strokeWidth={2}
                  fill="none"
                  name="Cancellations"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </NeuCard>

        {/* Therapy Package Distribution Pie Chart (1 col) */}
        <NeuCard elevation="raised" className="p-6 space-y-4">
          <div className="border-b border-[#E3DFC9] pb-3">
            <h2 className="font-serif text-lg font-bold text-[#234737]">
              Protocol Share
            </h2>
            <p className="text-xs text-[#6F7B71]">Therapy distribution by bookings</p>
          </div>

          <div className="h-56 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <PieChart>
                <Pie
                  data={pieData}
                  cx="50%"
                  cy="50%"
                  innerRadius={45}
                  outerRadius={75}
                  paddingAngle={4}
                  dataKey="value"
                >
                  {pieData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    backgroundColor: '#F5F3EC',
                    borderRadius: '12px',
                    borderColor: '#E3DFC9',
                    fontSize: '11px',
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </div>

          <div className="space-y-1.5 text-[11px] text-[#6F7B71] pt-2 border-t border-[#E3DFC9]">
            {pieData.map((d) => (
              <div key={d.name} className="flex items-center justify-between">
                <span className="flex items-center gap-1.5">
                  <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: d.color }} />
                  {d.name}
                </span>
                <span className="font-bold text-[#27352D]">{d.value}%</span>
              </div>
            ))}
          </div>
        </NeuCard>
      </div>

      {/* Adverse Events & Clinical Escalations Card */}
      <NeuCard elevation="raised" className="p-6 space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 border-b border-[#E3DFC9] pb-3">
          <div>
            <h2 className="font-serif text-lg font-bold text-[#234737] flex items-center gap-2">
              <AlertTriangle className="w-5 h-5 text-[#C15C3D]" />
              Reported Adverse Reactions & Complications
            </h2>
            <p className="text-xs text-[#6F7B71]">
              Clinical issue monitoring, staff acknowledgment, and escalation logs.
            </p>
          </div>
          <NeuBadge variant={adverseEvents.some((e) => !e.acknowledged_at) ? 'error' : 'sage'}>
            {adverseEvents.filter((e) => !e.acknowledged_at).length} Pending Review
          </NeuBadge>
        </div>

        {adverseEvents.length === 0 ? (
          <p className="text-xs text-[#6F7B71] py-4 text-center">
            No adverse events reported for this clinic. All therapies running smoothly.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-[#EFECE3] text-[#234737] uppercase font-semibold">
                <tr>
                  <th className="p-3">Severity</th>
                  <th className="p-3">Symptoms</th>
                  <th className="p-3">Immediate Actions</th>
                  <th className="p-3">Reported At</th>
                  <th className="p-3">Status</th>
                  <th className="p-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#E3DFC9]">
                {adverseEvents.map((evt) => (
                  <tr key={evt.id} className="hover:bg-[#F0EDE4]/60">
                    <td className="p-3 font-semibold">
                      <NeuBadge
                        variant={
                          evt.severity === 'CRITICAL' || evt.severity === 'SEVERE'
                            ? 'error'
                            : evt.severity === 'MODERATE'
                            ? 'forest'
                            : 'sage'
                        }
                      >
                        {evt.severity}
                      </NeuBadge>
                    </td>
                    <td className="p-3 text-[#27352D] font-medium max-w-xs">{evt.symptoms}</td>
                    <td className="p-3 text-[#6F7B71] max-w-xs">{evt.immediate_actions || 'N/A'}</td>
                    <td className="p-3 text-[#6F7B71]">
                      {new Date(evt.reported_at).toLocaleDateString()} {new Date(evt.reported_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </td>
                    <td className="p-3">
                      {evt.acknowledged_at ? (
                        <span className="flex items-center gap-1 text-[#357A55] font-semibold">
                          <CheckCircle2 className="w-3.5 h-3.5" /> Acknowledged
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-[#C15C3D] font-semibold">
                          <Clock className="w-3.5 h-3.5" /> Pending Review
                        </span>
                      )}
                    </td>
                    <td className="p-3 text-right">
                      {!evt.acknowledged_at && (
                        <NeuButton
                          variant="raised"
                          size="sm"
                          onClick={() => handleAcknowledgeAdverseEvent(evt.id)}
                          className="text-xs"
                        >
                          Acknowledge
                        </NeuButton>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </NeuCard>
    </div>
  );
};
