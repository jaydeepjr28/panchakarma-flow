import React, { useEffect, useState } from 'react';
import { NeuCard } from '@/components/ui/NeuCard';
import { NeuButton } from '@/components/ui/NeuButton';
import { NeuBadge } from '@/components/ui/NeuBadge';
import { NeuModal } from '@/components/ui/NeuModal';
import { NeuInput } from '@/components/ui/NeuInput';
import { NeuFeedbackStates } from '@/components/ui/NeuFeedbackStates';
import { roomApi } from '@/lib/api/client';
import { TreatmentRoom } from '@/lib/api/types';
import { useAuth } from '@/context/AuthContext';
import {
  DoorOpen,
  Plus,
  CheckCircle2,
  AlertCircle,
  Building2,
  Users,
} from 'lucide-react';

export const ClinicRoomsPage: React.FC = () => {
  const { user } = useAuth();
  const clinicId = user?.clinic_id || 'a3f8c872-9112-4c22-b5e1-7e8913c2f101';
  const [rooms, setRooms] = useState<TreatmentRoom[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // New Room Modal
  const [showAddModal, setShowAddModal] = useState(false);
  const [roomNumber, setRoomNumber] = useState('');
  const [roomName, setRoomName] = useState('');
  const [roomType, setRoomType] = useState('Snehana & Swedana');
  const [capacity, setCapacity] = useState(1);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [successMsg, setSuccessMsg] = useState('');

  const loadRooms = () => {
    setIsLoading(true);
    roomApi
      .listByClinic(clinicId)
      .then((data) => setRooms(data))
      .catch((err) => console.error(err))
      .finally(() => setIsLoading(false));
  };

  useEffect(() => {
    loadRooms();
  }, [clinicId]);

  const handleAddRoom = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    try {
      await roomApi.createRoom({
        clinic_id: clinicId,
        room_number: roomNumber,
        name: roomName,
        room_type: roomType,
        capacity,
      });
      setSuccessMsg(`Room "${roomName}" added to clinic inventory successfully.`);
      setShowAddModal(false);
      setRoomNumber('');
      setRoomName('');
      loadRooms();
    } catch (err: any) {
      alert(err.message || 'Failed to create room');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-widest font-bold text-[#C98769] mb-1">
            Facilities & Infrastructure
          </p>
          <h1 className="font-serif text-3xl font-bold text-[#234737]">
            Treatment Rooms & Therapy Suites
          </h1>
          <p className="text-xs text-[#6F7B71]">
            Specialized Droni suites, steam chambers, and room capacity management.
          </p>
        </div>

        <NeuButton
          variant="primary"
          size="md"
          onClick={() => setShowAddModal(true)}
          className="flex items-center gap-2"
        >
          <Plus className="w-4 h-4" /> Add Treatment Suite
        </NeuButton>
      </div>

      {successMsg && (
        <div className="p-4 rounded-xl bg-[#357A55]/10 border border-[#357A55]/20 text-xs text-[#357A55] flex items-center justify-between">
          <span>{successMsg}</span>
          <button onClick={() => setSuccessMsg('')} className="font-bold">✕</button>
        </div>
      )}

      {isLoading ? (
        <NeuFeedbackStates.Skeleton count={3} />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {rooms.map((room) => (
            <NeuCard key={room.id} elevation="raised" className="p-6 flex flex-col justify-between space-y-4">
              <div>
                <div className="flex items-start justify-between gap-2 mb-3">
                  <span className="text-xs font-bold px-2.5 py-1 rounded-full bg-[#315C49]/10 text-[#315C49]">
                    Suite #{room.room_number}
                  </span>
                  <NeuBadge variant={room.is_active ? 'success' : 'error'} size="sm">
                    {room.is_active ? 'In Service' : 'Maintenance'}
                  </NeuBadge>
                </div>

                <div className="flex items-center gap-2.5 mb-2">
                  <div className="w-10 h-10 rounded-xl bg-[#C98769]/15 text-[#C98769] flex items-center justify-center neu-inset shrink-0">
                    <DoorOpen className="w-5 h-5" />
                  </div>
                  <h2 className="font-serif text-base font-bold text-[#234737]">{room.name}</h2>
                </div>

                <div className="space-y-1.5 text-xs text-[#6F7B71] pt-3 border-t border-[#E3DFC9]">
                  <p>
                    <span className="font-semibold text-[#27352D]">Type:</span> {room.room_type}
                  </p>
                  <p>
                    <span className="font-semibold text-[#27352D]">Patient Capacity:</span> {room.capacity} patient(s) concurrently
                  </p>
                </div>
              </div>

              <div className="pt-3 border-t border-[#E3DFC9] flex justify-between items-center text-[11px] text-[#6F7B71]">
                <span>Equipped with Classical Wooden Droni</span>
                <span className="text-[#357A55] font-semibold">Ready</span>
              </div>
            </NeuCard>
          ))}
        </div>
      )}

      {/* Add Room Modal */}
      {showAddModal && (
        <NeuModal
          isOpen={showAddModal}
          onClose={() => setShowAddModal(false)}
          title="Add Treatment Room"
        >
          <form onSubmit={handleAddRoom} className="space-y-4">
            <NeuInput
              label="Room Number / Identifier"
              type="text"
              required
              value={roomNumber}
              onChange={(e) => setRoomNumber(e.target.value)}
              placeholder="e.g. 202"
            />

            <NeuInput
              label="Room Name"
              type="text"
              required
              value={roomName}
              onChange={(e) => setRoomName(e.target.value)}
              placeholder="e.g. Dhanwanthari Suite"
            />

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
                Specialized Therapy Type
              </label>
              <select
                value={roomType}
                onChange={(e) => setRoomType(e.target.value)}
                className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
              >
                <option value="Snehana & Swedana">Snehana & Swedana (Oleation & Steam)</option>
                <option value="Shirodhara Chamber">Shirodhara Chamber</option>
                <option value="Pizhichil & Kizhi Suite">Pizhichil & Kizhi Suite</option>
                <option value="Vamana / Virechana Kuti">Vamana / Virechana Kuti</option>
                <option value="Nasya & Mukhalepam Suite">Nasya & Mukhalepam Suite</option>
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-[#27352D] block mb-1.5">
                Simultaneous Patient Capacity
              </label>
              <input
                type="number"
                min={1}
                max={4}
                value={capacity}
                onChange={(e) => setCapacity(Number(e.target.value))}
                className="w-full px-4 py-2.5 rounded-xl bg-[#F5F3EC] neu-inset text-xs font-semibold text-[#27352D] focus:outline-none"
              />
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-[#E3DFC9]">
              <NeuButton variant="raised" onClick={() => setShowAddModal(false)}>
                Cancel
              </NeuButton>
              <NeuButton type="submit" variant="primary" isLoading={isSubmitting}>
                Save Suite
              </NeuButton>
            </div>
          </form>
        </NeuModal>
      )}
    </div>
  );
};
