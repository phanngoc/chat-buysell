import Head from 'next/head';
import { useRouter } from 'next/router';
import { useAuth } from '../components/AuthContext';

export default function Home() {
  const { user, login, logout, loading } = useAuth();
  const router = useRouter();

  // If user is logged in, redirect to chat
  if (!loading && user) {
    router.push('/chat');
    return null;
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50">
      <Head>
        <title>Chat Buy Sell App</title>
        <meta name="description" content="Connect buyers and sellers through chat" />
        <link rel="icon" href="/favicon.ico" />
      </Head>

      <main className="flex flex-col items-center justify-center w-full flex-1 px-4 sm:px-20 text-center">
        <h1 className="text-4xl font-bold text-gray-800 mb-6">
          Welcome to Chat Buy Sell
        </h1>
        
        <p className="text-xl text-gray-600 mb-8">
          Connect with buyers and sellers through our intelligent matching system
        </p>

        <div className="flex flex-col space-y-4">
          <button 
            onClick={login} 
            className="bg-[#1877F2] hover:bg-[#166FE5] text-white font-semibold py-3 px-6 rounded-md flex items-center justify-center"
          >
            <svg className="w-6 h-6 mr-2" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12.001 2.002c-5.522 0-9.999 4.477-9.999 9.999 0 4.99 3.656 9.126 8.437 9.879v-6.988h-2.54v-2.891h2.54V9.798c0-2.508 1.493-3.891 3.776-3.891 1.094 0 2.24.195 2.24.195v2.459h-1.264c-1.24 0-1.628.772-1.628 1.563v1.875h2.771l-.443 2.891h-2.328v6.988C18.344 21.129 22 16.992 22 12.001c0-5.522-4.477-9.999-9.999-9.999z"></path>
            </svg>
            Login with Facebook
          </button>
          
          <div className="text-sm text-gray-500 mt-4">
            By continuing, you agree to our Terms of Service and Privacy Policy
          </div>
        </div>
      </main>

      <footer className="w-full h-16 flex justify-center items-center border-t">
        <p className="text-sm text-gray-500">
          © {new Date().getFullYear()} Chat Buy Sell App
        </p>
      </footer>
    </div>
  );
}