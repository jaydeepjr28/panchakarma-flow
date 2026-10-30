const express = require('express');
const cors = require('cors');
const bcrypt = require('bcryptjs');
const jwt = require('jsonwebtoken');
const { v4: uuidv4 } = require('uuid');
const fs = require('fs-extra');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 3001;
const JWT_SECRET = process.env.JWT_SECRET || 'panchakarma_secret_key_2024';

// Enhanced CORS configuration
app.use(cors({
  origin: ['http://localhost:5173', 'http://localhost:3000', 'http://127.0.0.1:5173', 'http://localhost:4173'],
  credentials: true,
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization', 'X-Requested-With']
}));

// Middleware
app.use(express.json({ limit: '10mb' }));
app.use(express.urlencoded({ extended: true }));

// Add request logging
app.use((req, res, next) => {
  console.log(`${new Date().toISOString()} - ${req.method} ${req.path}`);
  if (req.body && Object.keys(req.body).length > 0) {
    console.log('Request body:', JSON.stringify(req.body, null, 2));
  }
  next();
});

// Data file paths
const DATA_DIR = path.join(__dirname, 'data');
const USERS_FILE = path.join(DATA_DIR, 'users.json');
const CLINICS_FILE = path.join(DATA_DIR, 'clinics.json');
const DOCTORS_FILE = path.join(DATA_DIR, 'doctors.json');
const PACKAGES_FILE = path.join(DATA_DIR, 'packages.json');
const APPOINTMENTS_FILE = path.join(DATA_DIR, 'appointments.json');
const SESSIONS_FILE = path.join(DATA_DIR, 'sessions.json');

// Ensure data directory and files exist
const initializeDataFiles = async () => {
  try {
    await fs.ensureDir(DATA_DIR);
    
    // Initialize empty files if they don't exist
    if (!await fs.pathExists(USERS_FILE)) {
      await fs.writeJson(USERS_FILE, []);
    }

    if (!await fs.pathExists(CLINICS_FILE)) {
      await fs.writeJson(CLINICS_FILE, []);
    }

    if (!await fs.pathExists(DOCTORS_FILE)) {
      await fs.writeJson(DOCTORS_FILE, []);
    }

    if (!await fs.pathExists(PACKAGES_FILE)) {
      await fs.writeJson(PACKAGES_FILE, []);
    }

    if (!await fs.pathExists(APPOINTMENTS_FILE)) {
      await fs.writeJson(APPOINTMENTS_FILE, []);
    }

    if (!await fs.pathExists(SESSIONS_FILE)) {
      await fs.writeJson(SESSIONS_FILE, []);
    }

    console.log('✅ Data files initialized successfully (empty)');
  } catch (error) {
    console.error('❌ Error initializing data files:', error);
  }
};

// Helper functions
const readJsonFile = async (filePath) => {
  try {
    return await fs.readJson(filePath);
  } catch (error) {
    console.error(`Error reading ${filePath}:`, error);
    return [];
  }
};

const writeJsonFile = async (filePath, data) => {
  try {
    await fs.writeJson(filePath, data, { spaces: 2 });
    return true;
  } catch (error) {
    console.error(`Error writing ${filePath}:`, error);
    return false;
  }
};

// Middleware to verify JWT token
const authenticateToken = (req, res, next) => {
  const authHeader = req.headers['authorization'];
  const token = authHeader && authHeader.split(' ')[1];

  if (!token) {
    return res.status(401).json({ error: 'Access token required' });
  }

  jwt.verify(token, JWT_SECRET, (err, user) => {
    if (err) {
      return res.status(403).json({ error: 'Invalid or expired token' });
    }
    req.user = user;
    next();
  });
};

// Validation functions
const validateEmail = (email) => {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
};

const validatePhone = (phone) => {
  if (!phone || typeof phone !== 'string') return false;
  const phoneRegex = /^[\+]?[1-9][\d]{0,15}$|^[\+]?[(]?[\d\s\-\(\)]{10,}$/;
  return phoneRegex.test(phone.replace(/[\s\-\(\)]/g, ''));
};

// Routes

// Health check
app.get('/api/health', (req, res) => {
  console.log('🏥 Health check requested');
  res.json({ 
    status: 'OK', 
    message: 'PanchakarmaFlow Backend is running',
    timestamp: new Date().toISOString(),
    cors: 'enabled'
  });
});

// Auth Routes
app.post('/api/auth/register', async (req, res) => {
  console.log('📝 Registration attempt for:', req.body.email);
  
  try {
    const { name, email, password, confirmPassword, phone, role } = req.body;

    // Validation
    if (!name?.trim()) {
      console.log('❌ Registration failed: Name required');
      return res.status(400).json({ error: 'Full name is required' });
    }

    if (!validateEmail(email)) {
      console.log('❌ Registration failed: Invalid email');
      return res.status(400).json({ error: 'Please enter a valid email address' });
    }

    if (!password || password.length < 6) {
      console.log('❌ Registration failed: Password too short');
      return res.status(400).json({ error: 'Password must be at least 6 characters long' });
    }

    if (password !== confirmPassword) {
      console.log('❌ Registration failed: Passwords do not match');
      return res.status(400).json({ error: 'Passwords do not match' });
    }

    if (!validatePhone(phone)) {
      console.log('❌ Registration failed: Invalid phone');
      return res.status(400).json({ error: 'Please enter a valid phone number' });
    }

    if (!role || !['patient', 'clinic_admin', 'therapist'].includes(role)) {
      console.log('❌ Registration failed: Invalid role');
      return res.status(400).json({ error: 'Please select a valid role' });
    }

    // Check if user already exists
    const users = await readJsonFile(USERS_FILE);
    const existingUser = users.find(u => u.email.toLowerCase() === email.toLowerCase());
    
    if (existingUser) {
      console.log('❌ Registration failed: Email already exists');
      return res.status(400).json({ error: 'An account with this email address already exists' });
    }

    // Create new user
    const hashedPassword = await bcrypt.hash(password, 10);
    const newUser = {
      id: uuidv4(),
      name: name.trim(),
      email: email.toLowerCase().trim(),
      password: hashedPassword,
      phone: phone.trim(),
      role,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      isActive: true,
      clinicId: undefined
    };

    users.push(newUser);
    await writeJsonFile(USERS_FILE, users);

    // Generate JWT token
    const token = jwt.sign(
      { 
        id: newUser.id, 
        email: newUser.email, 
        role: newUser.role 
      }, 
      JWT_SECRET, 
      { expiresIn: '7d' }
    );

    // Return user data (without password)
    const { password: _, ...userWithoutPassword } = newUser;
    
    console.log('✅ Registration successful for:', email);
    res.status(201).json({
      success: true,
      user: userWithoutPassword,
      token
    });

  } catch (error) {
    console.error('❌ Registration error:', error);
    res.status(500).json({ error: 'An unexpected error occurred during registration' });
  }
});

app.post('/api/auth/login', async (req, res) => {
  console.log('🔐 Login attempt for:', req.body.email);
  
  try {
    const { email, password } = req.body;

    // Validation
    if (!email || !password) {
      console.log('❌ Login failed: Missing credentials');
      return res.status(400).json({ error: 'Email and password are required' });
    }

    if (!validateEmail(email)) {
      console.log('❌ Login failed: Invalid email format');
      return res.status(400).json({ error: 'Please enter a valid email address' });
    }

    // Find user
    const users = await readJsonFile(USERS_FILE);
    const user = users.find(u => u.email.toLowerCase() === email.toLowerCase());

    if (!user) {
      console.log('❌ Login failed: User not found');
      return res.status(400).json({ error: 'Invalid email or password' });
    }

    if (!user.isActive) {
      console.log('❌ Login failed: Account deactivated');
      return res.status(400).json({ error: 'Account is deactivated. Please contact support.' });
    }

    // Verify password
    const isValidPassword = await bcrypt.compare(password, user.password);
    if (!isValidPassword) {
      console.log('❌ Login failed: Invalid password');
      return res.status(400).json({ error: 'Invalid email or password' });
    }

    // Update last login time
    user.lastLoginAt = new Date().toISOString();
    await writeJsonFile(USERS_FILE, users);

    // Generate JWT token
    const token = jwt.sign(
      { 
        id: user.id, 
        email: user.email, 
        role: user.role 
      }, 
      JWT_SECRET, 
      { expiresIn: '7d' }
    );

    // Return user data (without password)
    const { password: _, ...userWithoutPassword } = user;
    
    console.log('✅ Login successful for:', email);
    res.json({
      success: true,
      user: userWithoutPassword,
      token
    });

  } catch (error) {
    console.error('❌ Login error:', error);
    res.status(500).json({ error: 'An unexpected error occurred during login' });
  }
});

// User Routes
app.get('/api/users/me', authenticateToken, async (req, res) => {
  try {
    const users = await readJsonFile(USERS_FILE);
    const user = users.find(u => u.id === req.user.id && u.isActive);
    
    if (!user) {
      return res.status(404).json({ error: 'User not found' });
    }

    const { password: _, ...userWithoutPassword } = user;
    res.json({ user: userWithoutPassword });

  } catch (error) {
    console.error('Get user error:', error);
    res.status(500).json({ error: 'An unexpected error occurred' });
  }
});

app.get('/api/users', authenticateToken, async (req, res) => {
  try {
    const { role } = req.query;
    const users = await readJsonFile(USERS_FILE);
    
    let filteredUsers = users.filter(u => u.isActive);
    
    if (role) {
      filteredUsers = filteredUsers.filter(u => u.role === role);
    }

    // Remove passwords from response
    const usersWithoutPasswords = filteredUsers.map(({ password, ...user }) => user);
    
    res.json({ users: usersWithoutPasswords });

  } catch (error) {
    console.error('Get users error:', error);
    res.status(500).json({ error: 'An unexpected error occurred' });
  }
});

// Clinic Routes
app.get('/api/clinics', async (req, res) => {
  try {
    const clinics = await readJsonFile(CLINICS_FILE);
    console.log(`📋 Returning ${clinics.length} clinics`);
    res.json({ clinics });
  } catch (error) {
    console.error('Get clinics error:', error);
    res.status(500).json({ error: 'An unexpected error occurred' });
  }
});

app.get('/api/clinics/:id', async (req, res) => {
  try {
    const { id } = req.params;
    const clinics = await readJsonFile(CLINICS_FILE);
    const clinic = clinics.find(c => c.id === id);
    
    if (!clinic) {
      return res.status(404).json({ error: 'Clinic not found' });
    }

    res.json({ clinic });
  } catch (error) {
    console.error('Get clinic error:', error);
    res.status(500).json({ error: 'An unexpected error occurred' });
  }
});

app.post('/api/clinics', authenticateToken, async (req, res) => {
  try {
    const { name, address, phone, email, description, specialties } = req.body;

    // Only clinic_admin can create clinics
    if (req.user.role !== 'clinic_admin') {
      return res.status(403).json({ error: 'Only clinic administrators can create clinics' });
    }

    // Validation
    if (!name?.trim()) {
      return res.status(400).json({ error: 'Clinic name is required' });
    }

    if (!address?.trim()) {
      return res.status(400).json({ error: 'Clinic address is required' });
    }

    if (!validatePhone(phone)) {
      return res.status(400).json({ error: 'Please enter a valid phone number' });
    }

    if (!validateEmail(email)) {
      return res.status(400).json({ error: 'Please enter a valid email address' });
    }

    // Create new clinic
    const clinics = await readJsonFile(CLINICS_FILE);
    const newClinic = {
      id: uuidv4(),
      name: name.trim(),
      address: address.trim(),
      phone: phone.trim(),
      email: email.toLowerCase().trim(),
      description: description?.trim() || '',
      specialties: Array.isArray(specialties) ? specialties : [],
      rating: 0,
      image: 'https://images.unsplash.com/photo-1559757148-5c350d0d3c56?w=400&h=300&fit=crop',
      adminId: req.user.id,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };

    clinics.push(newClinic);
    await writeJsonFile(CLINICS_FILE, clinics);

    // Update user's clinicId
    const users = await readJsonFile(USERS_FILE);
    const userIndex = users.findIndex(u => u.id === req.user.id);
    if (userIndex !== -1) {
      users[userIndex].clinicId = newClinic.id;
      users[userIndex].updatedAt = new Date().toISOString();
      await writeJsonFile(USERS_FILE, users);
    }

    console.log('✅ Clinic created:', newClinic.name);
    res.status(201).json({
      success: true,
      clinic: newClinic
    });

  } catch (error) {
    console.error('Create clinic error:', error);
    res.status(500).json({ error: 'An unexpected error occurred while creating clinic' });
  }
});

// Doctor Routes
app.get('/api/doctors', async (req, res) => {
  try {
    const { clinicId } = req.query;
    const doctors = await readJsonFile(DOCTORS_FILE);
    
    let filteredDoctors = doctors;
    if (clinicId) {
      filteredDoctors = doctors.filter(d => d.clinicId === clinicId);
    }

    console.log(`👨‍⚕️ Returning ${filteredDoctors.length} doctors`);
    res.json({ doctors: filteredDoctors });
  } catch (error) {
    console.error('Get doctors error:', error);
    res.status(500).json({ error: 'An unexpected error occurred' });
  }
});

app.post('/api/doctors', authenticateToken, async (req, res) => {
  try {
    const { userId, clinicId } = req.body;

    // Only clinic_admin can add doctors
    if (req.user.role !== 'clinic_admin') {
      return res.status(403).json({ error: 'Only clinic administrators can add doctors' });
    }

    // Validation
    if (!userId?.trim()) {
      return res.status(400).json({ error: 'User ID is required' });
    }

    if (!clinicId?.trim()) {
      return res.status(400).json({ error: 'Clinic ID is required' });
    }

    // Check if user exists and is a therapist
    const users = await readJsonFile(USERS_FILE);
    const user = users.find(u => u.id === userId && u.role === 'therapist' && u.isActive);
    
    if (!user) {
      return res.status(404).json({ error: 'Therapist user not found' });
    }

    // Check if clinic exists and user is admin
    const clinics = await readJsonFile(CLINICS_FILE);
    const clinic = clinics.find(c => c.id === clinicId && c.adminId === req.user.id);
    
    if (!clinic) {
      return res.status(404).json({ error: 'Clinic not found or you are not authorized' });
    }

    // Check if doctor already exists
    const doctors = await readJsonFile(DOCTORS_FILE);
    const existingDoctor = doctors.find(d => d.userId === userId && d.clinicId === clinicId);
    
    if (existingDoctor) {
      return res.status(400).json({ error: 'This therapist is already added as a doctor in this clinic' });
    }

    // Create new doctor profile
    const newDoctor = {
      id: uuidv4(),
      userId: userId,
      name: user.name,
      qualifications: 'BAMS, MD (Ayurveda)', // Default, can be updated later
      specialties: ['Panchakarma', 'General Ayurveda'],
      experience: 5, // Default
      rating: 4.5, // Default
      clinicId: clinicId,
      image: 'https://images.unsplash.com/photo-1612349317150-e413f6a5b16d?w=150&h=150&fit=crop&crop=face',
      schedule: {
        monday: [{ start: '09:00', end: '17:00' }],
        tuesday: [{ start: '09:00', end: '17:00' }],
        wednesday: [{ start: '09:00', end: '17:00' }],
        thursday: [{ start: '09:00', end: '17:00' }],
        friday: [{ start: '09:00', end: '17:00' }],
        saturday: [{ start: '09:00', end: '13:00' }],
        sunday: []
      },
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };

    doctors.push(newDoctor);
    await writeJsonFile(DOCTORS_FILE, doctors);

    // Update user's clinicId
    const userIndex = users.findIndex(u => u.id === userId);
    if (userIndex !== -1) {
      users[userIndex].clinicId = clinicId;
      users[userIndex].updatedAt = new Date().toISOString();
      await writeJsonFile(USERS_FILE, users);
    }

    console.log('✅ Doctor added:', newDoctor.name);
    res.status(201).json({
      success: true,
      doctor: newDoctor
    });

  } catch (error) {
    console.error('Add doctor error:', error);
    res.status(500).json({ error: 'An unexpected error occurred while adding doctor' });
  }
});

// Package Routes
app.get('/api/packages', async (req, res) => {
  try {
    const { clinicId } = req.query;
    const packages = await readJsonFile(PACKAGES_FILE);
    
    let filteredPackages = packages;
    if (clinicId) {
      filteredPackages = packages.filter(p => p.clinicId === clinicId);
    }

    console.log(`📦 Returning ${filteredPackages.length} packages`);
    res.json({ packages: filteredPackages });
  } catch (error) {
    console.error('Get packages error:', error);
    res.status(500).json({ error: 'An unexpected error occurred' });
  }
});

app.post('/api/packages', authenticateToken, async (req, res) => {
  try {
    const { name, description, price, durationMinutes, category, clinicId, steps } = req.body;

    // Only clinic_admin can create packages
    if (req.user.role !== 'clinic_admin') {
      return res.status(403).json({ error: 'Only clinic administrators can create packages' });
    }

    // Validation
    if (!name?.trim()) {
      return res.status(400).json({ error: 'Package name is required' });
    }

    if (!description?.trim()) {
      return res.status(400).json({ error: 'Package description is required' });
    }

    if (!price || price <= 0) {
      return res.status(400).json({ error: 'Valid price is required' });
    }

    if (!durationMinutes || durationMinutes <= 0) {
      return res.status(400).json({ error: 'Valid duration is required' });
    }

    if (!category?.trim()) {
      return res.status(400).json({ error: 'Package category is required' });
    }

    if (!clinicId?.trim()) {
      return res.status(400).json({ error: 'Clinic ID is required' });
    }

    if (!steps || !Array.isArray(steps) || steps.length === 0) {
      return res.status(400).json({ error: 'At least one treatment step is required' });
    }

    // Verify clinic ownership
    const clinics = await readJsonFile(CLINICS_FILE);
    const clinic = clinics.find(c => c.id === clinicId && c.adminId === req.user.id);
    
    if (!clinic) {
      return res.status(403).json({ error: 'Clinic not found or you are not authorized' });
    }

    // Create new package
    const packages = await readJsonFile(PACKAGES_FILE);
    const newPackage = {
      id: uuidv4(),
      name: name.trim(),
      description: description.trim(),
      price: parseInt(price),
      durationMinutes: parseInt(durationMinutes),
      category: category.trim(),
      clinicId: clinicId.trim(),
      steps: steps.map(step => ({
        name: step.name?.trim(),
        duration: parseInt(step.duration) || 30,
        description: step.description?.trim() || ''
      })),
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };

    packages.push(newPackage);
    await writeJsonFile(PACKAGES_FILE, packages);

    console.log('✅ Package created:', newPackage.name);
    res.status(201).json({
      success: true,
      package: newPackage
    });

  } catch (error) {
    console.error('Create package error:', error);
    res.status(500).json({ error: 'An unexpected error occurred while creating package' });
  }
});

// Appointment Routes
app.get('/api/appointments', authenticateToken, async (req, res) => {
  try {
    const { patientId, clinicId, doctorId } = req.query;
    const appointments = await readJsonFile(APPOINTMENTS_FILE);
    
    let filteredAppointments = appointments;
    
    if (patientId) {
      filteredAppointments = filteredAppointments.filter(a => a.patientId === patientId);
    }
    
    if (clinicId) {
      filteredAppointments = filteredAppointments.filter(a => a.clinicId === clinicId);
    }
    
    if (doctorId) {
      filteredAppointments = filteredAppointments.filter(a => a.doctorId === doctorId);
    }

    console.log(`📅 Returning ${filteredAppointments.length} appointments`);
    res.json({ appointments: filteredAppointments });
  } catch (error) {
    console.error('Get appointments error:', error);
    res.status(500).json({ error: 'An unexpected error occurred' });
  }
});

app.post('/api/appointments', authenticateToken, async (req, res) => {
  try {
    const { clinicId, doctorId, packageId, date, time, notes } = req.body;
    const patientId = req.user.id;

    // Validation
    if (!clinicId || !doctorId || !packageId || !date || !time) {
      return res.status(400).json({ error: 'All appointment details are required' });
    }

    // Check if slot is available
    const appointments = await readJsonFile(APPOINTMENTS_FILE);
    const existingAppointment = appointments.find(a => 
      a.doctorId === doctorId && 
      a.date === date && 
      a.time === time &&
      ['pending', 'approved', 'confirmed'].includes(a.status)
    );

    if (existingAppointment) {
      return res.status(400).json({ error: 'This time slot is already booked' });
    }

    // Create new appointment
    const newAppointment = {
      id: uuidv4(),
      patientId,
      clinicId,
      doctorId,
      packageId,
      date,
      time,
      status: 'pending',
      notes: notes?.trim() || undefined,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };

    appointments.push(newAppointment);
    await writeJsonFile(APPOINTMENTS_FILE, appointments);

    console.log('✅ Appointment created:', newAppointment.id);
    res.status(201).json({
      success: true,
      appointment: newAppointment
    });

  } catch (error) {
    console.error('Create appointment error:', error);
    res.status(500).json({ error: 'An unexpected error occurred while booking appointment' });
  }
});

// Error handling middleware
app.use((err, req, res, next) => {
  console.error('❌ Server error:', err.stack);
  res.status(500).json({ error: 'Something went wrong!' });
});

// 404 handler
app.use((req, res) => {
  console.log('❌ Route not found:', req.path);
  res.status(404).json({ error: 'Route not found' });
});

// Initialize data and start server
const startServer = async () => {
  await initializeDataFiles();
  
  app.listen(PORT, '0.0.0.0', () => {
    console.log(`🚀 PanchakarmaFlow Backend Server running on port ${PORT}`);
    console.log(`📊 Health check: http://localhost:${PORT}/api/health`);
    console.log(`📁 Data directory: ${DATA_DIR}`);
    console.log(`🌐 CORS enabled for multiple origins`);
    console.log(`🏥 Empty data - only registered clinics and doctors will show`);
  });
};

startServer().catch(console.error);