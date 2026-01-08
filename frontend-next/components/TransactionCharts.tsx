'use client';

import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    BarElement,
    ArcElement,
    Title,
    Tooltip,
    Legend,
    Filler,
} from 'chart.js';
import { Line, Bar, Doughnut } from 'react-chartjs-2';

// Register Chart.js components
ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    BarElement,
    ArcElement,
    Title,
    Tooltip,
    Legend,
    Filler
);

interface ChartProps {
    type: 'line' | 'bar' | 'doughnut';
    data: { labels: string[]; data: number[] };
    color?: 'emerald' | 'purple' | 'blue';
}

const colorSchemes = {
    emerald: {
        borderColor: 'rgb(16, 185, 129)',
        backgroundColor: 'rgba(16, 185, 129, 0.2)',
    },
    purple: {
        borderColor: 'rgb(168, 85, 247)',
        backgroundColor: 'rgba(168, 85, 247, 0.2)',
    },
    blue: {
        borderColor: 'rgb(59, 130, 246)',
        backgroundColor: 'rgba(59, 130, 246, 0.2)',
    },
};

export default function TransactionCharts({ type, data, color = 'emerald' }: ChartProps) {
    const colors = colorSchemes[color];

    const commonOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                display: false,
            },
        },
        scales: type !== 'doughnut' ? {
            x: {
                grid: {
                    color: 'rgba(255, 255, 255, 0.1)',
                },
                ticks: {
                    color: 'rgba(255, 255, 255, 0.5)',
                },
            },
            y: {
                grid: {
                    color: 'rgba(255, 255, 255, 0.1)',
                },
                ticks: {
                    color: 'rgba(255, 255, 255, 0.5)',
                },
            },
        } : undefined,
    };

    if (type === 'line') {
        return (
            <Line
                data={{
                    labels: data.labels,
                    datasets: [{
                        data: data.data,
                        borderColor: colors.borderColor,
                        backgroundColor: colors.backgroundColor,
                        fill: true,
                        tension: 0.4,
                        pointRadius: 4,
                        pointBackgroundColor: colors.borderColor,
                    }],
                }}
                options={commonOptions}
            />
        );
    }

    if (type === 'bar') {
        return (
            <Bar
                data={{
                    labels: data.labels,
                    datasets: [{
                        data: data.data,
                        backgroundColor: colors.backgroundColor,
                        borderColor: colors.borderColor,
                        borderWidth: 1,
                        borderRadius: 4,
                    }],
                }}
                options={commonOptions}
            />
        );
    }

    if (type === 'doughnut') {
        return (
            <Doughnut
                data={{
                    labels: data.labels,
                    datasets: [{
                        data: data.data,
                        backgroundColor: [
                            'rgba(16, 185, 129, 0.8)',  // Success - green
                            'rgba(239, 68, 68, 0.8)',   // Failed - red
                            'rgba(234, 179, 8, 0.8)',   // Pending - yellow
                        ],
                        borderColor: [
                            'rgb(16, 185, 129)',
                            'rgb(239, 68, 68)',
                            'rgb(234, 179, 8)',
                        ],
                        borderWidth: 2,
                    }],
                }}
                options={{
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                color: 'rgba(255, 255, 255, 0.7)',
                                padding: 20,
                            },
                        },
                    },
                }}
            />
        );
    }

    return null;
}
