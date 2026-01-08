// Country coordinates for 3D globe visualization (lat/lng for capital cities)
export interface CountryGeo {
    code: string;
    name: string;
    lat: number;
    lng: number;
    currency: string;
}

// Top 50 GDP countries with their capital coordinates
export const COUNTRY_COORDINATES: CountryGeo[] = [
    { code: 'USA', name: 'United States', lat: 38.9072, lng: -77.0369, currency: 'USD' },
    { code: 'CHN', name: 'China', lat: 39.9042, lng: 116.4074, currency: 'CNY' },
    { code: 'DEU', name: 'Germany', lat: 52.5200, lng: 13.4050, currency: 'EUR' },
    { code: 'JPN', name: 'Japan', lat: 35.6762, lng: 139.6503, currency: 'JPY' },
    { code: 'IND', name: 'India', lat: 28.6139, lng: 77.2090, currency: 'INR' },
    { code: 'GBR', name: 'United Kingdom', lat: 51.5074, lng: -0.1278, currency: 'GBP' },
    { code: 'FRA', name: 'France', lat: 48.8566, lng: 2.3522, currency: 'EUR' },
    { code: 'ITA', name: 'Italy', lat: 41.9028, lng: 12.4964, currency: 'EUR' },
    { code: 'BRA', name: 'Brazil', lat: -15.7975, lng: -47.8919, currency: 'BRL' },
    { code: 'CAN', name: 'Canada', lat: 45.4215, lng: -75.6972, currency: 'CAD' },
    { code: 'RUS', name: 'Russia', lat: 55.7558, lng: 37.6173, currency: 'RUB' },
    { code: 'KOR', name: 'South Korea', lat: 37.5665, lng: 126.9780, currency: 'KRW' },
    { code: 'AUS', name: 'Australia', lat: -35.2809, lng: 149.1300, currency: 'AUD' },
    { code: 'MEX', name: 'Mexico', lat: 19.4326, lng: -99.1332, currency: 'MXN' },
    { code: 'ESP', name: 'Spain', lat: 40.4168, lng: -3.7038, currency: 'EUR' },
    { code: 'IDN', name: 'Indonesia', lat: -6.2088, lng: 106.8456, currency: 'IDR' },
    { code: 'NLD', name: 'Netherlands', lat: 52.3676, lng: 4.9041, currency: 'EUR' },
    { code: 'SAU', name: 'Saudi Arabia', lat: 24.7136, lng: 46.6753, currency: 'SAR' },
    { code: 'TUR', name: 'Turkey', lat: 39.9334, lng: 32.8597, currency: 'TRY' },
    { code: 'CHE', name: 'Switzerland', lat: 46.9480, lng: 7.4474, currency: 'CHF' },
    { code: 'POL', name: 'Poland', lat: 52.2297, lng: 21.0122, currency: 'PLN' },
    { code: 'TWN', name: 'Taiwan', lat: 25.0330, lng: 121.5654, currency: 'TWD' },
    { code: 'BEL', name: 'Belgium', lat: 50.8503, lng: 4.3517, currency: 'EUR' },
    { code: 'SWE', name: 'Sweden', lat: 59.3293, lng: 18.0686, currency: 'SEK' },
    { code: 'IRL', name: 'Ireland', lat: 53.3498, lng: -6.2603, currency: 'EUR' },
    { code: 'AUT', name: 'Austria', lat: 48.2082, lng: 16.3738, currency: 'EUR' },
    { code: 'THA', name: 'Thailand', lat: 13.7563, lng: 100.5018, currency: 'THB' },
    { code: 'ISR', name: 'Israel', lat: 31.7683, lng: 35.2137, currency: 'ILS' },
    { code: 'NGA', name: 'Nigeria', lat: 9.0579, lng: 7.4951, currency: 'NGN' },
    { code: 'ARE', name: 'United Arab Emirates', lat: 24.4539, lng: 54.3773, currency: 'AED' },
    { code: 'ARG', name: 'Argentina', lat: -34.6037, lng: -58.3816, currency: 'ARS' },
    { code: 'NOR', name: 'Norway', lat: 59.9139, lng: 10.7522, currency: 'NOK' },
    { code: 'EGY', name: 'Egypt', lat: 30.0444, lng: 31.2357, currency: 'EGP' },
    { code: 'VNM', name: 'Vietnam', lat: 21.0285, lng: 105.8542, currency: 'VND' },
    { code: 'BGD', name: 'Bangladesh', lat: 23.8103, lng: 90.4125, currency: 'BDT' },
    { code: 'ZAF', name: 'South Africa', lat: -25.7461, lng: 28.1881, currency: 'ZAR' },
    { code: 'PHL', name: 'Philippines', lat: 14.5995, lng: 120.9842, currency: 'PHP' },
    { code: 'DNK', name: 'Denmark', lat: 55.6761, lng: 12.5683, currency: 'DKK' },
    { code: 'MYS', name: 'Malaysia', lat: 3.1390, lng: 101.6869, currency: 'MYR' },
    { code: 'SGP', name: 'Singapore', lat: 1.3521, lng: 103.8198, currency: 'SGD' },
    { code: 'HKG', name: 'Hong Kong', lat: 22.3193, lng: 114.1694, currency: 'HKD' },
    { code: 'PAK', name: 'Pakistan', lat: 33.6844, lng: 73.0479, currency: 'PKR' },
    { code: 'CHL', name: 'Chile', lat: -33.4489, lng: -70.6693, currency: 'CLP' },
    { code: 'COL', name: 'Colombia', lat: 4.7110, lng: -74.0721, currency: 'COP' },
    { code: 'FIN', name: 'Finland', lat: 60.1699, lng: 24.9384, currency: 'EUR' },
    { code: 'CZE', name: 'Czech Republic', lat: 50.0755, lng: 14.4378, currency: 'CZK' },
    { code: 'ROU', name: 'Romania', lat: 44.4268, lng: 26.1025, currency: 'RON' },
    { code: 'PRT', name: 'Portugal', lat: 38.7223, lng: -9.1393, currency: 'EUR' },
    { code: 'NZL', name: 'New Zealand', lat: -41.2865, lng: 174.7762, currency: 'NZD' },
    { code: 'PER', name: 'Peru', lat: -12.0464, lng: -77.0428, currency: 'PEN' },
];

// Trade connections between countries (edges for Cytoscape graph)
export const TRADE_CONNECTIONS: [string, string][] = [
    // USD hub connections
    ['USA', 'GBR'], ['USA', 'DEU'], ['USA', 'JPN'], ['USA', 'CHN'], ['USA', 'CAN'],
    ['USA', 'MEX'], ['USA', 'AUS'], ['USA', 'CHE'], ['USA', 'KOR'], ['USA', 'IND'],
    ['USA', 'BRA'], ['USA', 'SGP'], ['USA', 'HKG'],
    // EUR connections
    ['DEU', 'FRA'], ['DEU', 'ITA'], ['DEU', 'ESP'], ['DEU', 'NLD'], ['DEU', 'BEL'],
    ['DEU', 'AUT'], ['DEU', 'POL'], ['DEU', 'CHE'], ['DEU', 'GBR'],
    ['FRA', 'ITA'], ['FRA', 'ESP'], ['FRA', 'BEL'], ['FRA', 'NLD'],
    // Asian connections
    ['CHN', 'JPN'], ['CHN', 'KOR'], ['CHN', 'HKG'], ['CHN', 'TWN'], ['CHN', 'SGP'],
    ['CHN', 'THA'], ['CHN', 'VNM'], ['CHN', 'MYS'], ['CHN', 'IDN'], ['CHN', 'IND'],
    ['JPN', 'KOR'], ['JPN', 'TWN'], ['JPN', 'SGP'], ['JPN', 'THA'],
    ['SGP', 'MYS'], ['SGP', 'HKG'], ['SGP', 'THA'], ['SGP', 'IDN'],
    // Middle East
    ['SAU', 'ARE'], ['SAU', 'EGY'], ['ARE', 'IND'],
    // South America
    ['BRA', 'ARG'], ['BRA', 'MEX'], ['BRA', 'CHL'], ['BRA', 'COL'],
    ['MEX', 'COL'], ['CHL', 'PER'], ['ARG', 'CHL'],
    // Africa
    ['ZAF', 'NGA'], ['ZAF', 'EGY'],
    // Oceania
    ['AUS', 'NZL'], ['AUS', 'SGP'], ['AUS', 'JPN'], ['AUS', 'CHN'],
    // Nordic
    ['SWE', 'NOR'], ['SWE', 'DNK'], ['SWE', 'FIN'], ['NOR', 'DNK'],
    // Eastern Europe
    ['POL', 'CZE'], ['CZE', 'AUT'], ['ROU', 'POL'],
    // Other major pairs
    ['GBR', 'IRL'], ['GBR', 'CHE'], ['GBR', 'IND'], ['GBR', 'HKG'],
    ['CHE', 'AUT'], ['ISR', 'USA'], ['TUR', 'DEU'],
];

// Get country by code
export function getCountryByCode(code: string): CountryGeo | undefined {
    return COUNTRY_COORDINATES.find(c => c.code === code);
}

// Get flag emoji from country code
export function getFlagEmoji(countryCode: string): string {
    const codeMap: Record<string, string> = {
        USA: 'US', CHN: 'CN', DEU: 'DE', JPN: 'JP', IND: 'IN', GBR: 'GB', FRA: 'FR',
        ITA: 'IT', BRA: 'BR', CAN: 'CA', RUS: 'RU', KOR: 'KR', AUS: 'AU', MEX: 'MX',
        ESP: 'ES', IDN: 'ID', NLD: 'NL', SAU: 'SA', TUR: 'TR', CHE: 'CH', POL: 'PL',
        TWN: 'TW', BEL: 'BE', SWE: 'SE', IRL: 'IE', AUT: 'AT', THA: 'TH', ISR: 'IL',
        NGA: 'NG', ARE: 'AE', ARG: 'AR', NOR: 'NO', EGY: 'EG', VNM: 'VN', BGD: 'BD',
        ZAF: 'ZA', PHL: 'PH', DNK: 'DK', MYS: 'MY', SGP: 'SG', HKG: 'HK', PAK: 'PK',
        CHL: 'CL', COL: 'CO', FIN: 'FI', CZE: 'CZ', ROU: 'RO', PRT: 'PT', NZL: 'NZ',
        PER: 'PE',
    };
    const code = codeMap[countryCode] || countryCode.slice(0, 2);
    const codePoints = [...code.toUpperCase()].map(c => 0x1F1E6 + c.charCodeAt(0) - 65);
    return String.fromCodePoint(...codePoints);
}
